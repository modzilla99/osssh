package openstack

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
	"github.com/modzilla99/osssh/types/openstack/nova"
)

func (c *OpenStackClient) GetNovaClient() (*gophercloud.ServiceClient, error) {
	return clientconfig.NewServiceClient(context.Background(), "compute", c.auth)
}

func getServerByID(c *gophercloud.ServiceClient, id string) (*nova.Server, error) {
	s := &nova.Server{}
	if err := servers.Get(context.TODO(), c, id).ExtractInto(s); err != nil {
		return nil, err
	}
	return s, nil
}

func getHypervisorFromServer(s *nova.Server) string {
	return s.HypervisorHostname
}