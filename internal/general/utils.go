package utils

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/hashicorp/go-uuid"
	"github.com/modzilla99/osssh/internal/ssh"
	"github.com/modzilla99/osssh/types/generic"
	gossh "golang.org/x/crypto/ssh"
)

func ParseArgs() (args generic.Args) {
	// args := generic.Args{}
	// Set username, default to the current username of the shell session
	username, _ := os.LookupEnv("USER")
	flag.StringVar(&args.Username, "u", username, "sets username to connect to HV with")

	flag.IntVar(&args.Port, "p", 2222, "Port for SSH to locally listen on")
	flag.IntVar(&args.RemotePort, "r", 22, "Remote port to forward traffic to")

	flag.Parse()
	parsedArgs := flag.Args()

	if args.Username == "" {
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

func bashGetHaProxyPid(net string) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
net='%s'
cmd="haproxy -f /var/lib/neutron/ovn-metadata-proxy/${net}.conf"

while read -r pid command; do
  if [[ "$command" = "$cmd" ]]; then
    printf '%s' "$pid"
    break
  fi
done < <(ps --no-headers -axo pid,command)`, net, "%d")
}

func GetNetNSFromNeutronMetadata(c *gossh.Client, net string) (string, error) {
	if strings.Contains(net, "'") || strings.HasSuffix(net, `\`) {
		return "", fmt.Errorf("invalid network id: %s", net)
	}
	fmt.Print("Obtaining path to NetworkNamespace...")

	out, stderr, err := ssh.RunCommand(c, bashGetHaProxyPid(net))
	if err != nil {
		return "", fmt.Errorf("unable to get pid of haproxy: stderr: %s error: %w", stderr, err)
	}

	fmt.Println("Done")
	return path.Join("/proc", out, "/ns/net"), nil
}
