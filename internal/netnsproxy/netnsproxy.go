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

func PortForwardViaSSH(c *gossh.Client, path string, address string, port int) (remotePid int) {
	fmt.Print("Setting up remote port forwarding...")
	remotePid, err := PortForward(c, path, 3022, generic.AddressPort{
		Address: address,
		Port: port,
		Type: "tcp",
	})
	if err != nil {
		fmt.Println()
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Done")
	return remotePid
}