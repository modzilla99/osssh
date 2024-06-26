package utils

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/hashicorp/go-uuid"
	"github.com/modzilla99/osssh/internal/ssh"
	"github.com/modzilla99/osssh/types/generic"
	gossh "golang.org/x/crypto/ssh"
)

func ParseArgs() (args generic.Args) {
	// args := generic.Args{}
	// Set username, default to the current username of the shell session
	username, gotUsernameFromEnv := os.LookupEnv("USERNAME")
	flag.StringVar(&args.Username, "u", username, "sets username to connect to HV with")

	flag.IntVar(&args.Port, "p", 2222, "Port for SSH to locally listen on")
	flag.IntVar(&args.RemotePort, "r", 22, "Remote port to forward traffic to")

	flag.Parse()
	parsedArgs := flag.Args()

	if !gotUsernameFromEnv && username == "" {
		fmt.Println("Cannot get username from environment, please specify username with -u")
		os.Exit(1)
	}

	if len(parsedArgs) != 1 {
		fmt.Println("Usage: osssh [-u] uuid")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if _, err := uuid.ParseUUID(parsedArgs[0]); err != nil {
		fmt.Println("Please specify a valid uuid for the server")
		os.Exit(1)
	}
	args.UUID = parsedArgs[0]
	return args
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