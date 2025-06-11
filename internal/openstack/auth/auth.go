package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/config/clouds"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
	"gopkg.in/yaml.v3"
)

type AuthOptions struct {
	AccessToken      string
	AuthType         clouds.AuthType
	AuthOptions      *gophercloud.AuthOptions
	IdentityProvider string
}

func NewAuthOptions(o *clientconfig.ClientOpts) (*AuthOptions, error) {
	ao, err := clientconfig.AuthOptions(o)
	if err != nil {
		return nil, err
	}

	// Parse clouds.yaml if environment variable is provided
	var cloud *Cloud
	if c := os.Getenv("OS_CLOUD"); c != "" {
		cloud, err = getAuthCloud(c)
		if err != nil {
			return nil, err
		}
	}

	// Set authentication params from clouds.yaml and Environment
	var (
		authType                      clouds.AuthType
		identityProvider, accessToken string
	)
	if cloud != nil {
		authType = cloud.AuthType
		identityProvider = cloud.IdentityProvider
		accessToken = cloud.AccessToken
	}

	authTypeEnv := os.Getenv(`OS_AUTH_TYPE`)
	identityProviderEnv := os.Getenv(`OS_IDENTITY_PROVIDER`)
	accessTokenEnv := os.Getenv(`OS_ACCESS_TOKEN`)

	// Enviroment variables override config options from file
	if authTypeEnv != "" {
		authType = clouds.AuthType(authTypeEnv)
	}
	if identityProviderEnv != "" {
		identityProvider = identityProviderEnv
	}
	if accessTokenEnv != "" {
		accessToken = accessTokenEnv
	}

	return &AuthOptions{AccessToken: accessToken, AuthOptions: ao, AuthType: authType, IdentityProvider: identityProvider}, nil
}

const Authv3OidcAccessToken clouds.AuthType = "v3oidcaccesstoken"

func Authenticate(o *clientconfig.ClientOpts) (provider *gophercloud.ProviderClient, err error) {
	ao, err := NewAuthOptions(o)
	if err != nil {
		return nil, err
	}

	// If neither domain name or domain id are set, set the name to default
	// Else if both are set, only use domain id
	if ao.AuthOptions.DomainName == "" && ao.AuthOptions.DomainID == "" {
		ao.AuthOptions.DomainName = "default"
	} else if ao.AuthOptions.DomainName != "" && ao.AuthOptions.DomainID != "" {
		ao.AuthOptions.DomainName = ""
	}

	if ao.AuthType == Authv3OidcAccessToken {
		return authenticateWithOIDC(ao)
	}

	provider, err = openstack.AuthenticatedClient(context.Background(), *ao.AuthOptions)
	if err != nil {
		return nil, err
	}

	return provider, nil
}

type Cloud struct {
	Cloud            string           `yaml:"cloud,omitempty" json:"cloud,omitempty"`
	Profile          string           `yaml:"profile,omitempty" json:"profile,omitempty"`
	AccessToken      string           `yaml:"access_token,omitempty" json:"access_token,omitempty"`
	AuthInfo         *clouds.AuthInfo `yaml:"auth,omitempty" json:"auth,omitempty"`
	AuthType         clouds.AuthType  `yaml:"auth_type,omitempty" json:"auth_type,omitempty"`
	RegionName       string           `yaml:"region_name,omitempty" json:"region_name,omitempty"`
	Regions          []clouds.Region  `yaml:"regions,omitempty" json:"regions,omitempty"`
	IdentityProvider string           `yaml:"identity_provider"`

	// EndpointType and Interface both specify whether to use the public, internal,
	// or admin interface of a service. They should be considered synonymous, but
	// EndpointType will take precedence when both are specified.
	EndpointType string `yaml:"endpoint_type,omitempty" json:"endpoint_type,omitempty"`
	Interface    string `yaml:"interface,omitempty" json:"interface,omitempty"`

	// API Version overrides.
	IdentityAPIVersion string `yaml:"identity_api_version,omitempty" json:"identity_api_version,omitempty"`
	VolumeAPIVersion   string `yaml:"volume_api_version,omitempty" json:"volume_api_version,omitempty"`

	// Verify whether or not SSL API requests should be verified.
	Verify *bool `yaml:"verify,omitempty" json:"verify,omitempty"`

	// CACertFile a path to a CA Cert bundle that can be used as part of
	// verifying SSL API requests.
	CACertFile string `yaml:"cacert,omitempty" json:"cacert,omitempty"`

	// ClientCertFile a path to a client certificate to use as part of the SSL
	// transaction.
	ClientCertFile string `yaml:"cert,omitempty" json:"cert,omitempty"`

	// ClientKeyFile a path to a client key to use as part of the SSL
	// transaction.
	ClientKeyFile string `yaml:"key,omitempty" json:"key,omitempty"`
}

func getAuthCloud(cloud string) (authType *Cloud, err error) {
	const (
		currentDir = "./clouds.yaml"
		userPath   = ".config/openstack/clouds.yaml"
		systemPath = "/etc/openstack/clouds.yaml"
	)
	f, err := os.ReadFile(currentDir)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	if err != nil && os.IsNotExist(err) {
		var homeDir string
		homeDir, err = os.UserHomeDir()
		f, err = os.ReadFile(path.Join(homeDir, userPath))
		if err != nil && !os.IsNotExist(err) {
			return
		}
		if err != nil && os.IsNotExist(err) {
			f, err = os.ReadFile(systemPath)
			if err != nil && !os.IsNotExist(err) {
				return
			}
			if err != nil && os.IsNotExist(err) {
				return nil, fmt.Errorf("Could not find clouds.yaml")
			}
		}
	}

	type Clouds struct {
		Clouds map[string]Cloud `yaml:"clouds" json:"clouds"`
	}

	var clouds Clouds
	if err := yaml.Unmarshal(f, &clouds); err != nil {
		return nil, err
	}
	c, ok := clouds.Clouds[cloud]
	if !ok {
		return nil, fmt.Errorf("Could not find cloud %s, in clouds.yaml", cloud)
	}

	return &c, nil
}

func authenticateWithOIDC(authOptions *AuthOptions) (provider *gophercloud.ProviderClient, err error) {
	if authOptions.IdentityProvider == "" ||
		authOptions.AuthOptions.IdentityEndpoint == "" ||
		authOptions.AccessToken == "" ||
		authOptions.AuthOptions.DomainName == "" ||
		authOptions.AuthOptions.TenantName == "" {
		return nil, errors.New("authentication not possbile, please provide all necessary environment/clouds.yaml configuration options")
	}

	// Issue unscoped token
	url := fmt.Sprintf("%s/OS-FEDERATION/identity_providers/%s/protocols/openid/auth", authOptions.AuthOptions.IdentityEndpoint, authOptions.IdentityProvider)
	osToken, err := issueUnscopedToken(url, authOptions.AccessToken)
	if err != nil {
		return nil, err
	}

	// use unscoped token for scoped token creation
	url = authOptions.AuthOptions.IdentityEndpoint + "/auth/tokens"
	token, catalog, err := issueScopedToken(url, osToken, authOptions.AuthOptions.DomainName, authOptions.AuthOptions.TenantName)
	if err != nil {
		return nil, err
	}

	o, err := clientconfig.AuthOptions(nil)
	if err != nil {
		return nil, err
	}
	o.TokenID = token
	o.DomainName = authOptions.AuthOptions.DomainName

	// Create provider client with issued token
	providerClient, err := openstack.NewClient(o.IdentityEndpoint)
	if err != nil {
		return nil, err
	}
	providerClient.TokenID = token
	mock := func(eo gophercloud.EndpointOpts) (string, error) {
		for _, e := range catalog {
			if e.Type != eo.Type {
				continue
			}

			for _, ep := range e.Endpoints {
				if ep.Interface == string(eo.Availability) {
					var url string
					if strings.HasSuffix(ep.URL, "/") {
						url = ep.URL
					} else {
						url = ep.URL + "/"
					}
					return url, nil
				}
			}
		}
		return "", errors.New("unsupported endpoint name: " + eo.Name)
	}
	providerClient.EndpointLocator = mock

	return providerClient, err
}

func issueUnscopedToken(url, bearer string) (string, error) {
	token := "Bearer " + bearer

	// Create a new HTTP request
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	// Create an HTTP client and send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return resp.Header.Get("X-Subject-Token"), nil
}

type Catalog []struct {
	Endpoints []struct {
		ID        string `json:"id"`
		Interface string `json:"interface"`
		RegionID  string `json:"region_id"`
		URL       string `json:"url"`
		Region    string `json:"region"`
	} `json:"endpoints"`
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

type Token struct {
	Token struct {
		Catalog Catalog `json:"catalog"`
	} `json:"token"`
}

func issueScopedToken(url, unscopedToken, domain, project string) (string, Catalog, error) {
	payload := map[string]map[string]map[string]interface{}{
		"auth": {
			"identity": {
				"methods": []string{
					"token",
				},
				"token": map[string]string{
					"id": unscopedToken,
				},
			},
			"scope": {
				"project": map[string]interface{}{
					"domain": map[string]string{
						"name": domain,
					},
					"name": project,
				},
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return "", nil, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error sending request:", err)
		return "", nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return "", nil, err
	}

	var catalog Token
	err = json.Unmarshal(body, &catalog)
	if err != nil {
		fmt.Println("Error decoding catalog:", err)
		return "", nil, err
	}

	return resp.Header.Get("X-Subject-Token"), catalog.Token.Catalog, nil
}
