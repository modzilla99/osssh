package ssh

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

type PortForwardSession struct {
	Listener         *net.Listener
	Remote           *net.Conn
	sessionOpen      bool
	initSessionClose bool
}

func newPortForwardSession(listener *net.Listener, remote *net.Conn) *PortForwardSession {
	return &PortForwardSession{
		Listener:         listener,
		Remote:           remote,
		sessionOpen:      true,
		initSessionClose: false,
	}
}

func (s *PortForwardSession) Close() error {
	(*s.Listener).Close()
	(*s.Remote).Close()
	s.initSessionClose = true
	for {
		if !s.sessionOpen {
			return nil
		}
	}
}

func PortForward(client *ssh.Client, port int, remoteAddress net.Addr) (*PortForwardSession, error) {
	sess, err := portForwardRetry(client, port, remoteAddress, 3)
	if err != nil {
		return nil, err
	}

	return sess, nil
}
func portForwardRetry(client *ssh.Client, port int, remoteAddress net.Addr, counter int) (*PortForwardSession, error) {
	if counter <= 0 {
		return nil, errors.New("too many retries")
	}
	var sess *PortForwardSession
	sess, err := portForward(client, port, remoteAddress)
	if err != nil {
		if err.Error() == "ssh: rejected: connect failed (Connection refused)" {
			fmt.Printf("Retrying...")
			time.Sleep(300 * time.Millisecond)
			sess, err = portForwardRetry(client, port, remoteAddress, counter-1)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return sess, nil
}

func portForward(client *ssh.Client, port int, remoteAddress net.Addr) (*PortForwardSession, error) {
	remote, err := client.Dial(remoteAddress.Network(), remoteAddress.String())
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, err
	}

	pfs := newPortForwardSession(
		&listener,
		&remote,
	)

	go func() {
		for {
			local, err := listener.Accept()
			if err != nil {
				if err.Error() == fmt.Sprintf("accept tcp 127.0.0.1:%d: use of closed network connection", port) {
					break
				}
				fmt.Println("Error", err)
				break
			}

			remote, err := client.Dial(remoteAddress.Network(), remoteAddress.String())
			if err != nil {
				fmt.Println("Error", err)
				break
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

			if pfs.initSessionClose {
				fmt.Println("Closed local port")
				break
			}
		}
		pfs.sessionOpen = false
	}()
	return pfs, nil
}
