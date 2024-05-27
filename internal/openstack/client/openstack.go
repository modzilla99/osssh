package openstack

import (
	"context"
	"fmt"
	"sync"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
	"github.com/modzilla99/osssh/types/openstack/neutron"
	"github.com/modzilla99/osssh/types/openstack/nova"
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
	var (
		wg sync.WaitGroup
		s *nova.Server
		serverPort *neutron.Port
		metadataPort *neutron.Port
		nova *gophercloud.ServiceClient
		neutron *gophercloud.ServiceClient
		err error
		errs []error
	)


	fmt.Print("Fetching data from OpenStack...")
	neutron, err = osc.GetNeutronClient()
	if err != nil {
		return nil, err
	}

	nova, err = osc.GetNovaClient()
	if err != nil {
		return nil, err
	}

	wg.Add(1)
	go func () {
		var er error
		defer wg.Done()
		s, er = getServerByID(nova, uuid)
		if er != nil {
			errs = append(errs, er)
		}
	}()

	wg.Add(1)
	go func () {
		var er error
		defer wg.Done()
		serverPort, er = getNeutronPortByServerID(neutron, uuid)
		if er != nil {
			errs = append(errs, er)
			return
		}
		metadataPort, er = getNeutronDistributedPortByNetworkID(neutron, serverPort.NetworkID)
		if er != nil {
			errs = append(errs, er)
		}
	}()

	wg.Wait()
	if len(errs) == 1 {
		return nil, errs[0]
	} else if len(errs) > 0 {
		fmt.Println("MultipleErrors")
		for _, err := range errs {
			fmt.Println(err)
		}
		return nil, errs[0]
	}

	fmt.Println("Done")
	return &Info{
		IPAddress: serverPort.FixedIPs[0].IPAddress,
		HypervisorHostname: getHypervisorFromServer(s),
		MetadataPort: metadataPort.DeviceID,
	}, nil
}
