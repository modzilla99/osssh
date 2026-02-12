package openstack

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
	"github.com/modzilla99/osssh/internal/openstack/auth"
	"github.com/modzilla99/osssh/types/openstack/neutron"
	"github.com/modzilla99/osssh/types/openstack/nova"
)

type Info struct {
	ServerName         string
	HypervisorHostname string
	IPAddress          string
	NetworkID          string
}

type OpenStackClient struct {
	ProviderClient *gophercloud.ProviderClient
	auth           *clientconfig.ClientOpts
}

func CreateClient(ctx context.Context) (*OpenStackClient, error) {
	fmt.Print("Authenticating to OpenStack...")
	opts := &clientconfig.ClientOpts{}
	provider, err := auth.Authenticate(ctx, opts)
	if err != nil {
		return nil, err
	}
	fmt.Println("Done")
	return &OpenStackClient{
		ProviderClient: provider,
		auth:           opts,
	}, nil
}

func GetInfo(ctx context.Context, osc *OpenStackClient, uuid string) (*Info, error) {
	var (
		wg         sync.WaitGroup
		s          *nova.Server
		serverPort *neutron.Port
		nova       *gophercloud.ServiceClient
		neutron    *gophercloud.ServiceClient
		err        error
		errs       []error
		cancel     context.CancelFunc
	)
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	fmt.Print("Fetching data from OpenStack...")
	neutron, err = osc.GetNeutronClient()
	if err != nil {
		return nil, err
	}

	nova, err = osc.GetNovaClient()
	if err != nil {
		return nil, err
	}

	wg.Go(func() {
		var e error
		s, e = getServerByID(nova, uuid)
		if e != nil {
			errs = append(errs, fmt.Errorf("getServerByID: %w", e))
		}
	})

	wg.Go(func() {
		var e error
		serverPort, e = getNeutronPortByServerID(ctx, neutron, uuid)
		if e != nil {
			errs = append(errs, fmt.Errorf("getNeutronPortByServerID: %w", e))
			return
		}
	})

	wg.Wait()
	if len(errs) == 1 {
		return nil, fmt.Errorf("experienced errors fetching data: %w", errs[0])
	} else if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	fmt.Println("Done")
	return &Info{
		ServerName:         s.Name,
		HypervisorHostname: s.HypervisorHostname,
		IPAddress:          serverPort.FixedIPs[0].IPAddress,
		NetworkID:          serverPort.NetworkID,
	}, nil
}
