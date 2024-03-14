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

var remotePids []int
var hypervisor string

func main() {
	uuid, username, port, remotePort := utils.ParseArgs()
	setupCleanup()
	osc, err := openstack.CreateClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	i, _ := openstack.GetInfo(osc, uuid)
	fmt.Printf("Obtained the following information %#v\n", i)

	fmt.Print("Connecting to SSH...")
	c, err := ssh.NewClient(i.HypervisorHostname, username)
	if err != nil {
		fmt.Printf("Error\n%s\n", err)
		os.Exit(1)
	}
	defer c.Close()
	fmt.Println("Done")

	pid := utils.GetPidOfNeutronMetadata(c)
	netnsproxy.Setup(c)

	netns := fmt.Sprintf("/proc/%d/root/run/netns/%s", pid, i.MetadataPort)
	proxyPort, remotePid := netnsproxy.PortForwardViaSSH(c, netns, i.IPAddress, remotePort)
	remotePids = append(remotePids, remotePid)

	fmt.Print("Setting up local port forwarding...")
	pfs, err := ssh.PortForward(c, port, generic.AddressPort{
		Address: "127.0.0.1",
		Port: proxyPort,
		Type: "tcp",
	})
	if err != nil {
		fmt.Printf("Error\n%s\n", err)
		os.Exit(1)
	}
	defer pfs.Close()
	fmt.Printf("Done\nForwarding %s:22 from netns %s to 127.0.0.1:%d\n", i.IPAddress, netns, port)

	cha := make(chan struct{})
	<-cha
}

func cleanup() {
    c, err := ssh.NewClient(hypervisor, "jlamp")
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