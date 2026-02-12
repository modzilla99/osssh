package openstack

import (
	"context"
	"errors"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/modzilla99/osssh/types/openstack/neutron"
)

func (c *OpenStackClient) GetNeutronClient() (*gophercloud.ServiceClient, error) {
	return openstack.NewNetworkV2(c.ProviderClient, gophercloud.EndpointOpts{})
}

func getNeutronPortByServerID(ctx context.Context, c *gophercloud.ServiceClient, id string) (*neutron.Port, error) {
	s := ports.ListOpts{
		DeviceID: id,
		Limit:    1,
	}
	p, err := ports.List(c, s).AllPages(ctx)
	if err != nil {
		return nil, err
	}

	ps := make([]neutron.Port, 0, 1)
	err = ports.ExtractPortsInto(p, &ps)
	if err != nil {
		return nil, err
	}

	if len(ps) == 0 {
		return nil, errors.New("no port found for server with id: " + id)
	}
	return &ps[0], nil
}
