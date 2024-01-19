package utils

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/modzilla99/osssh/internal/ssh"
	gossh "golang.org/x/crypto/ssh"
)

func ParseArgs() (uuid string, username string) {
	for _, arg := range os.Args {
		if arg == "-h" {
			help()
		}
		if arg == "--help" {
			help()
		}
	}
	if len(os.Args) > 4 {
		fmt.Println("Too many arguments provided")
		os.Exit(1)
	}
	if len(os.Args) < 2 {
		fmt.Println("Please provide UUID of Server as an argument")
		os.Exit(1)
	}
	for seq, arg := range os.Args {
		if len(arg) == 36 {
			uuid = arg
			continue
		}
		if arg == "-u" || arg == "--user" {
			if len(os.Args) == seq + 2 && len(uuid) == 0 || len(os.Args) == seq + 1 {
				fmt.Println("Please specify a username")
				os.Exit(1)
			}
			username = os.Args[seq + 1]
		}
	}
	fmt.Print(len(uuid))
	if len(uuid) == 0 {
		fmt.Println("Please specify a uuid")
	}
	return uuid, username
}

func help() {
	fmt.Println(`usage: osssh [-h] uuid

Port-Forward SSH to a server without FloatingIP via Metadata-Port.
	
positional arguments:
    uuid                UUID of server to SSH into

options:
-h, --help            show this help message and exit
-u, --user username   sets username to connect to HV with`)
	os.Exit(0)
}

func GetPidOfNeutronMetadata(c *gossh.Client) (pid int) {
	fmt.Print("Obtaining PID of Neutron Metadata Server...")
	out, stderr, err := ssh.RunCommand(c, "sudo docker inspect -f '{{.State.Pid}}' neutron-metadata-agent-ovn")
	if err != nil {
		log.Fatalln(stderr, err)
	}
	pid, err = strconv.Atoi(out)
	if err != nil {
		fmt.Println()
		log.Fatalln(err)
	}
	fmt.Println("Done")
	fmt.Printf("Got PID of Neutron Metadata Agent: %d\n", pid)
	return pid
}