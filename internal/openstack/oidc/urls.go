package oidc

import "github.com/gophercloud/gophercloud/v2"

// "%s/OS-FEDERATION/identity_providers/%s/protocols/openid/auth",
func authURLOpenID(c *gophercloud.ServiceClient, idp string) string {
	return c.ServiceURL("OS-FEDERATION", "identity_providers", idp, "protocols", "openid", "auth")
}

func authURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("auth", "tokens")
}
