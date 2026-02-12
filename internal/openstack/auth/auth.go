package auth

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/config/clouds"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
	"github.com/modzilla99/osssh/internal/openstack/oidc"
	gophercloudOidc "github.com/modzilla99/osssh/internal/openstack/oidc"
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

func Authenticate(ctx context.Context, o *clientconfig.ClientOpts) (provider *gophercloud.ProviderClient, err error) {
	ao, err := NewAuthOptions(o)
	if err != nil {
		return nil, err
	}

	if ao.AuthType == Authv3OidcAccessToken {
		return oidc.AuthenticatedClient(ctx, &gophercloudOidc.AuthOptions{
			AccessToken:      ao.AccessToken,
			IdentityProvider: ao.IdentityProvider,
			IdentityEndpoint: ao.AuthOptions.IdentityEndpoint,
			DomainID:         ao.AuthOptions.DomainID,
			DomainName:       ao.AuthOptions.DomainName,
			Scope:            tokens.Scope(*ao.AuthOptions.Scope),
		})
	}

	provider, err = openstack.AuthenticatedClient(ctx, *ao.AuthOptions)
	if err != nil {
		return nil, err
	}

	return provider, nil
}

type Cloud struct {
	AccessToken      string          `yaml:"access_token,omitempty" json:"access_token,omitempty"`
	AuthType         clouds.AuthType `yaml:"auth_type,omitempty" json:"auth_type,omitempty"`
	IdentityProvider string          `yaml:"identity_provider,omitempty" json:"identity_provider,omitempty"`
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
