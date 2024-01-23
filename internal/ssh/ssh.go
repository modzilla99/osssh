package ssh

import (
	"bytes"
	"encoding/base64"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

func NewClient(hostname string, username string) (*ssh.Client, error) {
	// get hostkey of remote host to verify legitimacy
	hostKey, err := knownhosts.New(filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"))
	if err != nil {
		return nil, err
	}

	s, err := ConnectSSHAgentSock()
	if err != nil {
		return nil, err
	}
	defer (*s).Close()
	a, err := ConnectSSHAgent(s)
	if err != nil {
		return nil, err
	}

	pkAuth, err := GetPrivateKeys(a)
	if err != nil {
		return nil, err
	}

	// Authentication
    config := &ssh.ClientConfig{
        User: username,
        Auth: []ssh.AuthMethod{pkAuth},
        HostKeyCallback: hostKey,
		HostKeyAlgorithms: []string{
			ssh.KeyAlgoED25519,
			ssh.KeyAlgoRSASHA512,
			ssh.KeyAlgoRSASHA256,
		},
    }
    // Connect
    client, err := ssh.Dial("tcp", net.JoinHostPort(hostname, "22"), config)
    if err != nil {
        return nil, err
    }

	return client, nil
}

func GetSession(client *ssh.Client) (*ssh.Session, error) {
	// Create a session. It is one session per command.
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	// defer session.Close()
	return session, err
}

func RunCommand(client *ssh.Client, cmd string) (string, string, error) {
	s, err := GetSession(client)
	if err != nil {
		return "", "", err
	}
	defer s.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	s.Stdout = &stdout
	s.Stderr = &stderr

	err = s.Run(cmd)
	return strings.TrimSuffix(stdout.String(), "\n"), strings.TrimSuffix(stderr.String(), "\n"), err
}

type SshBackgroundTask struct {
	session *ssh.Session
}

func (t *SshBackgroundTask) Close() error {
	t.session.Signal(ssh.SIGTERM)
	t.session.Close()
	return nil
}

func RunCommandBackground(client *ssh.Client, cmd string) (*SshBackgroundTask, error) {
	s, err := GetSession(client)
	if err != nil {
		return nil, err
	}
	// s.Shell()
	go func() {
		s.Run(cmd)
	}()

	return &SshBackgroundTask{session: s}, nil
}

func WriteFile(client *ssh.Client, fileName string, file []byte) (error) {
	stdin := base64.StdEncoding.EncodeToString(file)
	s, err := GetSession(client)
	if err != nil {
		return err
	}
	defer s.Close()

	s.Stdin = bytes.NewBufferString(stdin)

	s.Run("base64 -d > " + fileName)
	return nil
}

func ConnectSSHAgentSock() (*net.Conn, error) {
	agentPath, exist := os.LookupEnv("SSH_AUTH_SOCK")
	if ! exist {
		log.Fatalln("SSH-Agent is not running, unable to authenticate...")
	}
	// Connect to ssh-agent socket
	agentSock, err := net.Dial("unix", agentPath)
	if err != nil {
		return nil, err
	}
	return &agentSock, err
}

func ConnectSSHAgent(s *net.Conn) (agent.ExtendedAgent, error) {
	return agent.NewClient(*s), nil
}

func GetPrivateKeys(a agent.ExtendedAgent) (ssh.AuthMethod, error) {
	signers, err := a.Signers()
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signers...), nil
}