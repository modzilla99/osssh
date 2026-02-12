package oidc

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

type AuthOptions struct {
	AccessToken      string
	IdentityProvider string
	IdentityEndpoint string
	UnscopedTokenID  string
	Scope            tokens.Scope
	DomainID         string
	DomainName       string
}

func (opts AuthOptions) ToTokenV3CreateMap(scope map[string]any) (map[string]any, error) {
	gophercloudAuthOpts := gophercloud.AuthOptions{
		DomainID:    opts.DomainID,
		DomainName:  opts.DomainName,
		AllowReauth: false,
		TokenID:     opts.UnscopedTokenID,
	}

	return gophercloudAuthOpts.ToTokenV3CreateMap(scope)
}

// ToTokenV3ScopeMap builds a scope request body from AuthOptions.
func (opts *AuthOptions) ToTokenV3ScopeMap() (map[string]any, error) {
	scope := gophercloud.AuthScope(opts.Scope)

	gophercloudAuthOpts := gophercloud.AuthOptions{
		Scope:      &scope,
		DomainID:   opts.DomainID,
		DomainName: opts.DomainName,
	}

	return gophercloudAuthOpts.ToTokenV3ScopeMap()
}

func (opts AuthOptions) ToTokenV3HeadersMap(headerOpts map[string]any) (map[string]string, error) {
	return nil, nil
}

func (opts *AuthOptions) CanReauth() bool {
	return false
}

// Create authenticated Token
func Create(ctx context.Context, client *gophercloud.ServiceClient, opts *AuthOptions) (r tokens.CreateResult) {
	if opts.UnscopedTokenID == "" {
		resp, err := client.Post(ctx, authURLOpenID(client, opts.IdentityProvider), nil, &r.Body, &gophercloud.RequestOpts{
			MoreHeaders: map[string]string{
				"Authorization": "Bearer " + opts.AccessToken,
			},
			OkCodes: []int{
				201,
			},
		})
		_, headers, err := gophercloud.ParseResponse(resp, err)
		if err != nil {
			r.Err = err
			return
		}

		opts.UnscopedTokenID = headers.Get("X-Subject-Token")
	}
	// opts.DomainID = ""
	opts.DomainName = ""

	return tokens.Create(ctx, client, opts)
}
