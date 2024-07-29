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
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

func Authenticate(o *clientconfig.ClientOpts) (provider *gophercloud.ProviderClient, err error) {
	provider, err = clientconfig.AuthenticatedClient(context.Background(), o)
	if err != nil {
		if err.Error() == "You must provide exactly one of DomainID or DomainName in a Scope with ProjectName" {
			o.AuthInfo.DomainName = "default"

			return clientconfig.AuthenticatedClient(context.Background(), o)
		}
		if os.Getenv("OS_PROTOCOL") == "openid" {
			return authenticateWithOIDC()
		}
		return nil, err
	}
	return provider, nil
}


func authenticateWithOIDC() (provider *gophercloud.ProviderClient, err error) {
	//identityProvider := "nws-id"
	identityProvider := os.Getenv("OS_IDENTITY_PROVIDER")
	//keystoneURL := "https://cloud.netways.de:5000/v3"
	keystoneURL := strings.TrimSuffix(os.Getenv("OS_AUTH_URL"), "/")
	// Keystone JWT token
	token := os.Getenv("OS_ACCESS_TOKEN")
	// OpenStack Project Domain Name used for creation of scoped token
	projectDomainName := os.Getenv("OS_PROJECT_DOMAIN_NAME")
	// OpenStack Project Name used as the project for the token
	projectName := os.Getenv("OS_PROJECT_NAME")


	if identityProvider == "" || keystoneURL == "" || token == "" || projectDomainName == "" || projectName == "" {
		fmt.Println("Please provide OS_IDENTITY_PROVIDER, OS_AUTH_URL, OS_ACCESS_TOKEN, OS_PROJECT_DOMAIN_NAME and OS_PROJECT_NAME as enviroment variables")
		return nil, errors.New("authentication impossible")
	}

	// Issue unscoped token
	url := fmt.Sprintf("%s/OS-FEDERATION/identity_providers/%s/protocols/openid/auth", keystoneURL, identityProvider)
	osToken, err := issueUnscopedToken(url, token)
	if err != nil {
		return nil, err
	}

	// use unscoped token for scoped token creation
	url = keystoneURL + "/auth/tokens"
	token, catalog, err := issueScopedToken(url, osToken, projectDomainName, projectName)
	if err != nil {
		return nil, err
	}

	o, err := clientconfig.AuthOptions(nil)
	if err != nil {
		return nil, err
	}
	o.TokenID = token
	o.DomainName = projectDomainName

	// Create provider client with issued token
	providerClient, err := openstack.NewClient(o.IdentityEndpoint)
	if err != nil {
		return nil, err
	}
	providerClient.TokenID = token
	mock := func (eo gophercloud.EndpointOpts) (string, error) {
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