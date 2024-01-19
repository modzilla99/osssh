package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	utils "github.com/modzilla99/osssh/internal/general"
	"github.com/modzilla99/osssh/internal/netnsproxy"
	openstack "github.com/modzilla99/osssh/internal/openstack/client"
	"github.com/modzilla99/osssh/internal/ssh"
	"github.com/modzilla99/osssh/types/generic"
)

type Process interface {
	Close() error
}

var processes []Process
var remotePids []int
var hypervisor string

func main() {
	uuid, username := utils.ParseArgs()
	setupCleanup()
	osc, err := openstack.CreateClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	i, _ := openstack.GetInfo(osc, uuid)
	fmt.Printf("Obtained the following information %#v\n", i)

	fmt.Print("Connecting to SSH...")
	hypervisor = i.HypervisorHostname
	c, err := ssh.NewClient(hypervisor, username)
	if err != nil {
		fmt.Printf("Error\n%s\n", err)
		os.Exit(1)
	}
	fmt.Println("Done")

	pid := utils.GetPidOfNeutronMetadata(c)
	netnsproxy.Setup(c)

	netns := fmt.Sprintf("/proc/%d/root/run/netns/%s", pid, i.MetadataPort)
	remotePids = append(remotePids, netnsproxy.PortForwardViaSSH(c, netns, i.IPAddress, 22))

	fmt.Print("Setting up local port forwarding...")
	pfs, err := ssh.PortForward(c, 2222, generic.AddressPort{
		Address: "127.0.0.1",
		Port: 3022,
		Type: "tcp",
	})
	if err != nil {
		log.Fatalln(err)
	}
	processes = append(processes, pfs)
	fmt.Println("Done")
	fmt.Printf("Forwarding %s:22 from netns %s to 127.0.0.1:2222\n", i.IPAddress, netns)

	cha := make(chan struct{})
	<-cha
}

func setupCleanup() {
	cleaner := make(chan os.Signal, 1)
    signal.Notify(cleaner, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-cleaner
		fmt.Println("Cleaning up")
		for _, p := range processes {
			if err := p.Close(); err != nil {
				fmt.Printf("Error closing process: %s\n", err.Error())
			} else {
				fmt.Println("Successfully cleaned up process")
			}
		}

		c, err := ssh.NewClient(hypervisor, "jlamp")
		if err != nil {
			fmt.Printf("Error\n%s\n", err)
			os.Exit(1)
		}
		for _, pid := range remotePids {
			_, stderr, err := ssh.RunCommand(c, fmt.Sprintf(`sudo kill -TERM %d`, pid))
			if err != nil {
				fmt.Printf("Error cleaning up remote netns-proxy process: %d\n", pid)
				if strings.Contains(stderr, "No such process") {
					_, stderr, err = ssh.RunCommand(c, `sudo killall -TERM /tmp/netns-proxy`)
					if strings.Contains(stderr, "no process found") {
						fmt.Println("It seems like all netns-proxies are already stopped")
						os.Exit(0)
					}
					if err != nil {
						fmt.Println(err)
						os.Exit(1)
					}
					fmt.Println("Killed all remaining netns-proxy processes")
					os.Exit(0)
				}
			}
		}
        os.Exit(0)
    }()
}