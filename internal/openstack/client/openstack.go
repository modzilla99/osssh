package openstack

import (
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/modzilla99/osssh/types/openstack/neutron"
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
	provider, err = clientconfig.AuthenticatedClient(o)
	if err != nil {
		if err.Error() == "You must provide exactly one of DomainID or DomainName in a Scope with ProjectName" {
			o.AuthInfo.DomainName = "default"
			return clientconfig.AuthenticatedClient(o)
		}
		return nil, err
	}
	return provider, nil
}

func (c *OpenStackClient) GetNeutronClient() (*gophercloud.ServiceClient, error) {
	n, err := clientconfig.NewServiceClient("network", c.auth)
	if err != nil {
		fmt.Println("Error getting Neutron client: ", err)
	}
	return n, err
}

func (c *OpenStackClient) GetNovaClient() (*gophercloud.ServiceClient, error) {
	return clientconfig.NewServiceClient("compute", c.auth)
}

func GetInfo(osc *OpenStackClient, uuid string) (*Info, error) {
	fmt.Print("Fetching data from OpenStack...")
	n, _ := osc.GetNeutronClient()
	pS, _ := getNeutronPortByServerID(n, uuid)
	pD, _ := getNeutronDistributedPortByNetworkID(n, pS.NetworkID)

	fmt.Println("Done")
	return &Info{
		IPAddress: pS.FixedIPs[0].IPAddress,
		HypervisorHostname: pS.HostID,
		MetadataPort: pD.DeviceID,
	}, nil
}

func getNeutronPortByServerID(c *gophercloud.ServiceClient, id string) (*neutron.Port, error) {
	s := ports.ListOpts{
		DeviceID: id,
	}
	p, _ := ports.List(c, s).AllPages()
	ap, _ := extractPorts(p)
	if len(ap) == 0 {
		return nil, fmt.Errorf("no ports found")
	}
	return &ap[0], nil
}

func getNeutronDistributedPortByNetworkID(c *gophercloud.ServiceClient, id string) (*neutron.Port, error) {
	s := ports.ListOpts{
		NetworkID: id,
		DeviceOwner: "network:distributed",
	}
	pa, _ := ports.List(c, s).AllPages()
	p, _ := extractPorts(pa)
	if len(p) == 0 {
		return nil, fmt.Errorf("unable to retrieve distributed port")
	}
	return &p[0], nil
}

func extractPorts(r pagination.Page) ([]neutron.Port, error) {
	var s []neutron.Port
	err := ports.ExtractPortsInto(r, &s)
	return s, err
}