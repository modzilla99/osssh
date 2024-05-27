package openstack

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

type Info struct {
	HypervisorHostname string
	IPAddress string
	MetadataPort string
} 

type OpenStackClient struct {
	ProviderClient *gophercloud.ProviderClient
	auth *clientconfig.ClientOpts
}

func CreateClient() (*OpenStackClient, error) {
	fmt.Print("Authenticating to OpenStack...")
	opts := new(clientconfig.ClientOpts)
	provider, err := authenticate(opts)
	if err != nil {
		return nil, err
	}
	fmt.Println("Done")
	return &OpenStackClient{
		ProviderClient: provider,
		auth: opts,
	}, nil
}

func authenticate(o *clientconfig.ClientOpts) (provider *gophercloud.ProviderClient, err error) {
	provider, err = clientconfig.AuthenticatedClient(context.Background(), o)
	if err != nil {
		if err.Error() == "You must provide exactly one of DomainID or DomainName in a Scope with ProjectName" {
			o.AuthInfo.DomainName = "default"
			return clientconfig.AuthenticatedClient(context.Background(), o)
		}
		return nil, err
	}
	return provider, nil
}

func GetInfo(osc *OpenStackClient, uuid string) (*Info, error) {
	fmt.Print("Fetching data from OpenStack...")
	neutron, err := osc.GetNeutronClient()
	if err != nil {
		return nil, err
	}

	nova, err := osc.GetNovaClient()
	if err != nil {
		return nil, err
	}

	s, err := getServerByID(nova, uuid)
	if err != nil {
		return nil, err
	}

	pS, err := getNeutronPortByServerID(neutron, uuid)
	if err != nil {
		return nil, err
	}
	pD, err := getNeutronDistributedPortByNetworkID(neutron, pS.NetworkID)
	if err != nil {
		return nil, err
	}

	fmt.Println("Done")
	return &Info{
		IPAddress: pS.FixedIPs[0].IPAddress,
		HypervisorHostname: getHypervisorFromServer(s),
		MetadataPort: pD.DeviceID,
	}, nil
}
