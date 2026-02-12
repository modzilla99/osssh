package ssh

import (
	"context"
	"fmt"
	"io"
	"net"

	"golang.org/x/crypto/ssh"
)

func PortForward(ctx context.Context, client *ssh.Client, port int, remoteAddress net.Addr) error {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		var local net.Conn
		newConn := make(chan struct{})
		go func() {
			local, err = listener.Accept()
			close(newConn)
		}()

		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled")
		case <-newConn:
			if err != nil {
				if err.Error() == fmt.Sprintf("accept tcp 127.0.0.1:%d: use of closed network connection", port) {
					break
				}
				fmt.Println("Error", err)
				return err
			}
		}

		err = handleNewConnection(client, local, remoteAddress)
		if err != nil {
			fmt.Println("Error", err)
			break
		}
	}

	return nil
}

func handleNewConnection(client *ssh.Client, local net.Conn, remoteAddress net.Addr) error {
	remote, err := client.Dial(remoteAddress.Network(), remoteAddress.String())
	if err != nil {
		return err
	}
	done := make(chan struct{}, 2)

	go func() {
		io.Copy(local, remote)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(remote, local)
		done <- struct{}{}
	}()
	return nil
}
