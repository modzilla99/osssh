package netnsproxy

import (
	"context"
	"embed"
	"fmt"
	"time"

	"github.com/modzilla99/osssh/internal/ssh"
	gossh "golang.org/x/crypto/ssh"
)

var (
	//go:embed files
	files embed.FS
)

func Setup(c *gossh.Client) error {
	fmt.Print("Uploading netns-proxy...")
	file, _ := GetNetnsProxyFileBytes(true)
	_, _, err := ssh.RunCommand(c, "test -e /tmp/netns-proxy")
	if err == nil {
		fmt.Println("Ok")
		return nil
	}

	err = ssh.WriteFile(c, "/tmp/netns-proxy", file)
	if err != nil {
		return fmt.Errorf("Unable to copy netnsproxy to host: %w", err)
	}

	_, _, err = ssh.RunCommand(c, "chmod +x /tmp/netns-proxy")
	if err != nil {
		return fmt.Errorf("cannot set permissions of netnsproxy: %w", err)
	}

	fmt.Println("Done")
	return nil
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


type NetnsProxyOpts struct {
	Path string
	Address string
	ListenPort int
	ProxyPort int
}

const bashWrapper = `run_me() {
  %s
}

run_me &
pid=$!
trap "kill -INT $PID || kill -TERM $PID" INT TERM
wait $PID
`

func (o NetnsProxyOpts) Command () string {
	exec := fmt.Sprintf("/usr/bin/sudo /tmp/netns-proxy -b 127.0.0.1:%d -p %s %s %s:%d",
		o.ListenPort, "tcp", o.Path, o.Address, o.ProxyPort,
	)
	return fmt.Sprintf(bashWrapper, exec)
}

func RunNetnsProxy(ctx context.Context, client *gossh.Client, opts NetnsProxyOpts) error {
	sess, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("unable to open session: %w", err)
	}
	defer sess.Close()

	err = sess.RequestPty("xterm", 80, 40, gossh.TerminalModes{gossh.ECHO: 0})
	if err != nil {
		return fmt.Errorf("unable to request pty: %w", err)
	}

	err = sess.Start(opts.Command())
	if err != nil {
		return fmt.Errorf("cannot start netnsproxy: %w", err)
	}

	waitChan := make(chan error, 1)
	go func() {
		waitChan <- sess.Wait()
	}()

	select {
	case <-waitChan:
		return fmt.Errorf("netnsproxy exited unexpectedly")

	case <-ctx.Done():
		fmt.Print("Shutting down remote netnsproxy...")

		err = sess.Signal(gossh.SIGINT)
		if err != nil {
			return fmt.Errorf("unable to send interrupt to remote process: %w", err)
		}

		timeout := time.NewTicker(5 * time.Second)
		defer timeout.Stop()

		select {
		case err = <-waitChan:
			if err != nil {
				switch t := err.(type) {
				case *gossh.ExitError:
					if t.ExitStatus() == 130 {
						return nil
					}
				default:
				}
				return fmt.Errorf("error waiting for process to finish: %w", err)
			}
		case <-timeout.C:
			return fmt.Errorf("timeout reached stopping netsproxy")
		}
	}
	return nil
}

func checkPortAvailability(c *gossh.Client, port int) (bool, error) {
	_, _, err := ssh.RunCommand(c, fmt.Sprintf("nc -z 127.0.0.1 %d", port))
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

func GetAvailablePort(c *gossh.Client) (proxyPort int, err error) {
	var portOk bool
	const (
		proxyPortStart = 3021 // + 1
		proxyPortEnd   = 3052
	)

	proxyPort = proxyPortStart
	for !portOk {
		proxyPort = proxyPort + 1

		if proxyPort >= proxyPortEnd {
			err = fmt.Errorf("cannot find available port between: %d - %d", proxyPortStart, proxyPortEnd)
			break
		}

		portOk, err = checkPortAvailability(c, proxyPort)
		if err != nil {
			err = fmt.Errorf("unable to check for available ports: %w\n", err)
			break
		}
	}
	return proxyPort, err
}


func PortForwardToNetns(ctx context.Context, doneCh chan struct{}, c *gossh.Client, opts NetnsProxyOpts) {
	fmt.Print("Setting up remote port forwarding...")

	// Get next available port as multiple netnsproxy instances might be running on remote
	err := RunNetnsProxy(ctx, c, opts)
	if err != nil {
		fmt.Println(err)
		close(doneCh)
		return
	}

	close(doneCh)
}
