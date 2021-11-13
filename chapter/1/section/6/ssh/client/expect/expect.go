package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/user"
	"regexp"
	"strings"
	"time"

	"github.com/google/goexpect"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	private = flag.String("private", "", "The path to the SSH private key for this connection")
)

func main() {
	flag.Parse()

	if len(os.Args) != 2 {
		fmt.Println("Error: command must be 1 arg, [host]")
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

	fmt.Println("Installing Expect on remote system")

	if err := installExpect(conn); err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}

	fmt.Println("Done")
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

func installExpect(conn *ssh.Client) (err error) {
	// Here we are setting up an io.Writer that will write
	// to our debug strings.Builder{}. If we run into an
	// error, the output of the command will dumpt to STDERR.
	r, w := io.Pipe()
	debug := strings.Builder{}
	debugDone := make(chan struct{})
	go func() {
		io.Copy(&debug, r)
		close(debugDone)
	}()

	defer func() {
		// Wait for our io.Copy() to be done.
		<-debugDone

		// Only log this if we had an error.
		if err != nil {
			log.Printf("expect debug:\n%s", debug.String())
		}
	}()

	e, _, err := expect.SpawnSSH(conn, 5*time.Second, expect.Tee(w))
	if err != nil {
		return err
	}
	defer e.Close()

	var promptRE = regexp.MustCompile(`\$ `)

	_, _, err = e.Expect(promptRE, 10*time.Second)
	if err != nil {
		return fmt.Errorf("did not get shell prompt")
	}

	if err := e.Send("sudo apt-get install expect\n"); err != nil {
		return fmt.Errorf("error on send command: %s", err)
	}

	_, _, ecase, err := e.ExpectSwitchCase(
		[]expect.Caser{
			&expect.Case{
				R: regexp.MustCompile(`Do you want to continue\? \[Y/n\] `),
				T: expect.OK(),
			},
			&expect.Case{
				R: regexp.MustCompile(`is already the newest`),
				T: expect.OK(),
			},
		},
		10*time.Second,
	)
	if err != nil {
		return fmt.Errorf("apt-get install did not send what we expected")
	}

	switch ecase {
	case 0:
		if err := e.Send("Y\n"); err != nil {
			return err
		}
	}

	_, _, err = e.Expect(promptRE, 10*time.Second)
	if err != nil {
		return fmt.Errorf("did not get shell prompt")
	}

	return nil
}
