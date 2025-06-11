package netnsproxy

import (
	"embed"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/modzilla99/osssh/internal/ssh"
	"github.com/modzilla99/osssh/types/generic"
	gossh "golang.org/x/crypto/ssh"
)

var (
	//go:embed files
	files embed.FS
)

func Setup(c *gossh.Client) {
	fmt.Print("Uploading netns-proxy...")
	file, _ := GetNetnsProxyFileBytes(true)
	_, _, err := ssh.RunCommand(c, "test -e /tmp/netns-proxy")
	if err != nil {
		ssh.WriteFile(c, "/tmp/netns-proxy", file)
		ssh.RunCommand(c, "chmod +x /tmp/netns-proxy")
		fmt.Println("Done")
	} else {
		fmt.Println("Ok")
	}
}

func GetNetnsProxyFileBytes(old bool) ([]byte, error) {

	var fileName string
	if old {
		fileName = "files/netns-proxy-glic-old"
	} else {
		fileName = "files/netns-proxy-glic-new"
	}
	return files.ReadFile(fileName)
}

func PortForward(client *gossh.Client, netns string, localPort int, remote net.Addr) (int, error) {
	cmd := fmt.Sprintf(
		`sudo bash -c "nohup /tmp/netns-proxy -p %s -b 127.0.0.1:%d %s %s > /dev/null 2>&1 & jobs -p"`,
		remote.Network(), localPort, netns, remote.String())
	stdout, stderr, err := ssh.RunCommand(client, cmd)
	// time.Sleep(time.Second)
	if err != nil {
		fmt.Println("Error")
		return 0, err
	}
	if stderr != "" {
		fmt.Println("Error")
		return 0, fmt.Errorf(stderr)
	}
	pid, err := strconv.Atoi(stdout)
	if err != nil {
		fmt.Println("Error")
		return 0, err
	}
	return pid, nil
}

func checkPortAvailability(c *gossh.Client, port int) (available bool, err error) {
	_, _, err = ssh.RunCommand(c, fmt.Sprintf("nc -z 127.0.0.1 %d", port))
	if err != nil {
		switch err := err.(type) {
		case *gossh.ExitError:
			if err.ExitStatus() == 1 {
				return true, nil
			}
			return false, err
		default:
			return false, err
		}
	}
	return false, nil
}

func PortForwardViaSSH(c *gossh.Client, path string, address string, port int) (proxyPort int, remotePid int) {
	fmt.Print("Setting up remote port forwarding...")

	// Get next available port as multiple netnsproxy instances might be running on remote
	proxyPort = 3022
	ok := false
	var err error
	for !ok {
		if proxyPort >= 4000 {
			fmt.Println("Error")
			fmt.Println("Unable to find available port on remote...")
			os.Exit(1)
		}
		ok, err = checkPortAvailability(c, proxyPort)
		if err != nil {
			fmt.Printf("Error\nError checking port availability for port %d\n%s", proxyPort, err.Error())
			os.Exit(1)
		}
		proxyPort = proxyPort + 1
	}

	remotePid, err = PortForward(c, path, proxyPort, generic.AddressPort{
		Address: address,
		Port:    port,
		Type:    "tcp",
	})
	if err != nil {
		fmt.Println()
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Done")
	return proxyPort, remotePid
}
