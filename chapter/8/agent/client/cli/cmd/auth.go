package cmd

import (
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func agentAuth() (ssh.AuthMethod, error) {
	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil, err
	}

	client := agent.NewClient(conn)
	return ssh.PublicKeysCallback(client.Signers), nil
}

func publicKey(privateKeyFile string) (ssh.AuthMethod, error) {
	k, err := os.ReadFile(privateKeyFile)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(k)
	if err != nil {
		return nil, err
	}

	return ssh.PublicKeys(signer), nil
}

func getAuthFromFlags() (ssh.AuthMethod, error) {
	if keyFile != "" {
		return publicKey(keyFile)
	}
	return agentAuth()
}
