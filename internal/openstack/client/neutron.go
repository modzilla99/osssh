package openstack

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
	"github.com/modzilla99/osssh/types/openstack/neutron"
)

func (c *OpenStackClient) GetNeutronClient() (*gophercloud.ServiceClient, error) {
	n, err := clientconfig.NewServiceClient(context.Background(), "network", c.auth)
	if err != nil {
		fmt.Println("Error getting Neutron client: ", err)
	}
	return n, err
}

func getNeutronPortByServerID(c *gophercloud.ServiceClient, id string) (*neutron.Port, error) {
	s := ports.ListOpts{
		DeviceID: id,
	}
	p, err := ports.List(c, s).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	ap, err := extractPorts(p)
	if err != nil {
		return nil, err
	}
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
	pa, err := ports.List(c, s).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	p, err := extractPorts(pa)
	if err != nil {
		return nil, err
	}
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