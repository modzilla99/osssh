package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	utils "github.com/modzilla99/osssh/internal/general"
	"github.com/modzilla99/osssh/internal/netnsproxy"
	openstack "github.com/modzilla99/osssh/internal/openstack/client"
	"github.com/modzilla99/osssh/internal/ssh"
	"github.com/modzilla99/osssh/types/generic"
)

type Process interface {
	Close() error
}

var (
	hypervisor string
	username   string
)

func main() {
	args := utils.ParseArgs()
	username = args.Username
	ctx := context.Background()
	osc, err := openstack.CreateClient(ctx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	i, err := openstack.GetInfo(ctx, osc, args.UUID)
	if err != nil {
		fmt.Printf("Error\n%s\n", err)
		os.Exit(1)
	}
	hypervisor = i.HypervisorHostname

	if err := run(ctx, i, args); err != nil {
		fmt.Println("Error")
		fmt.Printf("Error %s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, info *openstack.Info, args generic.Args) error {
	fmt.Print("Connecting to SSH...")
	c, err := ssh.NewClient(hypervisor, args.Username)
	if err != nil {
		fmt.Printf("Error\n%s\n", err)
		os.Exit(1)
	}
	defer c.Close()
	fmt.Println("Done")

	path, err := utils.GetNetNSFromNeutronMetadata(c, info.NetworkID)
	if err != nil {
		return err
	}

	err = netnsproxy.Setup(c)
	if err != nil {
		return err
	}

	proxyPort, err := netnsproxy.GetAvailablePort(c)
	if err != nil {
		return err
	}

	doneChan := make(chan struct{})
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	setupCleanup(cancel, doneChan)

	go netnsproxy.PortForwardToNetns(ctx, doneChan, c, netnsproxy.NetnsProxyOpts{
		ListenPort: proxyPort,
		Address: info.IPAddress,
		Path: path,
		ProxyPort: args.RemotePort,
	})

	time.Sleep(200 *time.Millisecond)
	select {
	case <-doneChan:
		return fmt.Errorf("failed to setup port-forwarding")
	default:
		println("Done")
	}

	fmt.Print("Setting up local port forwarding...")
	pfs, err := ssh.PortForward(c, args.Port, generic.AddressPort{
		Address: "127.0.0.1",
		Port:    proxyPort,
		Type:    "tcp",
	})
	if err != nil {
		select {
		case <-doneChan:
			return err
		default:
			close(doneChan)
		}
		return err
	}
	defer pfs.Close()
	fmt.Printf("Done\nForwarding %s:%d (%s on %s) from network %s to 127.0.0.1:%d\n",
		info.IPAddress, args.RemotePort, info.ServerName, info.HypervisorHostname, info.NetworkID, args.Port)

	<-doneChan
	return nil
}

func setupCleanup(cancel context.CancelFunc, done chan struct{}) {
	cleaner := make(chan os.Signal, 1)
	signal.Notify(cleaner, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-cleaner
		cancel()
		<-done
		os.Exit(0)
	}()
}
