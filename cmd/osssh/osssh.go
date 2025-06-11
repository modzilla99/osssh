package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	utils "github.com/modzilla99/osssh/internal/general"
	"github.com/modzilla99/osssh/internal/netnsproxy"
	openstack "github.com/modzilla99/osssh/internal/openstack/client"
	"github.com/modzilla99/osssh/internal/ssh"
	"github.com/modzilla99/osssh/types/generic"
	gossh "golang.org/x/crypto/ssh"
)

type Process interface {
	Close() error
}

var (
	remotePids []int
	hypervisor string
	username   string
)

func main() {
	args := utils.ParseArgs()
	username = args.Username
	setupCleanup()
	osc, err := openstack.CreateClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	i, err := openstack.GetInfo(osc, args.UUID)
	if err != nil {
		fmt.Printf("Error\n%s\n", err)
		os.Exit(1)
	}
	hypervisor = i.HypervisorHostname

	if err := run(i, args); err != nil {
		fmt.Println("Error")
		cleanup()
		fmt.Printf("Error %s\n", err)
		os.Exit(1)
	}
}

func run(info *openstack.Info, args generic.Args) error {
	fmt.Print("Connecting to SSH...")
	c, err := ssh.NewClient(hypervisor, args.Username)
	if err != nil {
		fmt.Printf("Error\n%s\n", err)
		os.Exit(1)
	}
	defer c.Close()
	fmt.Println("Done")

	pid := utils.GetPidOfNeutronMetadata(c)
	netnsproxy.Setup(c)

	netns := fmt.Sprintf("/proc/%d/root/run/netns/%s", pid, info.MetadataPort)
	proxyPort, remotePid := netnsproxy.PortForwardViaSSH(c, netns, info.IPAddress, args.RemotePort)
	remotePids = append(remotePids, remotePid)

	fmt.Print("Setting up local port forwarding...")
	pfs, err := ssh.PortForward(c, args.Port, generic.AddressPort{
		Address: "127.0.0.1",
		Port:    proxyPort,
		Type:    "tcp",
	})
	if err != nil {
		return err
	}
	defer pfs.Close()
	fmt.Printf("Done\nForwarding %s:22 from netns %s to 127.0.0.1:%d\n", info.IPAddress, netns, args.Port)

	cha := make(chan struct{})
	<-cha
	return nil
}

func cleanup() {
	c, err := ssh.NewClient(hypervisor, username)
	if err != nil {
		fmt.Printf("Error creating SSH client: %s\n", err)
		panic(err.Error())
	}
	defer c.Close()

	fmt.Println("Cleaning up...")
	cleanupRemoteProcesses(c)
	fmt.Println("Cleanup complete.")
}

func cleanupRemoteProcesses(c *gossh.Client) {
	if len(remotePids) == 0 {
		fmt.Println("No remote process found, trying to kill all remote processes")
		cleanupAllRemainingRemoteProcesses(c)
		return
	}
	for _, pid := range remotePids {
		_, stderr, err := ssh.RunCommand(c, fmt.Sprintf(`sudo kill -TERM %d`, pid))
		if err != nil {
			fmt.Printf("Error cleaning up all remote netns-proxy processes: %d\n%s\n", pid, err)
			if strings.Contains(stderr, "No such process") {
				cleanupAllRemainingRemoteProcesses(c)
				return
			}
		}
	}
}

func cleanupAllRemainingRemoteProcesses(c *gossh.Client) {
	_, stderr, err := ssh.RunCommand(c, `sudo killall -TERM /tmp/netns-proxy`)
	if strings.Contains(stderr, "no process found") {
		fmt.Println("All netns-proxies are already stopped")
		return
	}
	if err != nil {
		fmt.Println("Error killing remaining netns-proxy processes")
		panic(err.Error())
	}
	fmt.Println("Killed all remaining netns-proxy processes")
}

func setupCleanup() {
	cleaner := make(chan os.Signal, 1)
	signal.Notify(cleaner, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-cleaner
		cleanup()
		os.Exit(0)
	}()
}
