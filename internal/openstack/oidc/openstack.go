package oidc

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
)

func AuthenticatedClient(ctx context.Context, authopts *AuthOptions) (*gophercloud.ProviderClient, error) {
	client, err := openstack.NewClient(authopts.IdentityEndpoint)
	if err != nil {
		return nil, err
	}
	client.ReauthFunc = nil

	keystone, err := openstack.NewIdentityV3(client, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	result := Create(ctx, keystone, authopts)
	err = client.SetTokenAndAuthResult(result)
	if err != nil {
		return nil, err
	}

	catalog, err := result.ExtractServiceCatalog()
	if err != nil {
		return nil, fmt.Errorf("unable to extract service catalog from token: %w", err)
	}

	client.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		return openstack.V3EndpointURL(catalog, opts)
	}
	return client, nil
}
