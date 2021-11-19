package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/user"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	private = flag.String("private", "", "The path to the SSH private key for this connection")
)

func main() {
	flag.Parse()

	if len(os.Args) != 3 {
		fmt.Println("Error: command must be 2 args, [host] [command]")
		os.Exit(1)
	}
	_, _, err := net.SplitHostPort(os.Args[1])
	if err != nil {
		os.Args[1] = os.Args[1] + ":22"
		_, _, err = net.SplitHostPort(os.Args[1])
		if err != nil {
			fmt.Println("Error: problem with host passed: ", err)
			os.Exit(1)
		}
	}

	var auth ssh.AuthMethod

	if *private == "" {
		fi, _ := os.Stdin.Stat()
		if (fi.Mode() & os.ModeCharDevice) == 0 {
			fmt.Println("-private not set, cannot use password when STDIN is a pipe")
			os.Exit(1)
		}
		auth, err = passwordFromTerm()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		auth, err = publicKey(*private)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	u, err := user.Current()
	if err != nil {
		fmt.Println("Error: problem getting current user: ", err)
		os.Exit(1)
	}

	config := &ssh.ClientConfig{
		User:            u.Username,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	conn, err := ssh.Dial("tcp", os.Args[1], config)
	if err != nil {
		fmt.Println("Error: could not dial host: ", err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	out, err := combinedOutput(ctx, conn, os.Args[2])
	if err != nil {
		fmt.Println("command error: ", err)
		os.Exit(1)
	}
	fmt.Println(out)
}

func passwordFromTerm() (ssh.AuthMethod, error) {
	fmt.Printf("SSH Passsword: ")
	p, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}
	fmt.Println("") // Show the return
	if len(bytes.TrimSpace(p)) == 0 {
		return nil, fmt.Errorf("password was an empty string")
	}
	return ssh.Password(string(p)), nil
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

// combinedOutput runs a command on an SSH client. The context can be cancelled, however
// SSH does not always honor the kill signals we send, so this might not break. So closing
// the session does nothing. So depending on what the server is doing, cancelling the context
// may do nothing and it may still block.
func combinedOutput(ctx context.Context, conn *ssh.Client, cmd string) (string, error) {
	sess, err := conn.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()

	if v, ok := ctx.Deadline(); ok {
		t := time.NewTimer(v.Sub(time.Now()))
		defer t.Stop()

		go func() {
			x := <-t.C
			if !x.IsZero() {
				sess.Signal(ssh.SIGKILL)
			}
		}()
	}

	b, err := sess.Output(cmd)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
