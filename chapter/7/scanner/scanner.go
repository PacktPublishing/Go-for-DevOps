package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/user"
	"sync"
	"time"

	"inet.af/netaddr"
)

const (
	ping = "ping"
	ssh  = "ssh"

	uname = "uname"
)

func main() {
	_, err := exec.LookPath(ping)
	if err != nil {
		log.Fatal("cannot find ping in our PATH")
	}
	_, err = exec.LookPath(ssh)
	if err != nil {
		log.Fatal("cannot find ssh in our PATH")
	}

	if len(os.Args) != 2 {
		log.Fatal("error: only one argument allowed, the network CIDR to scan")
	}

	ipCh, err := hosts(os.Args[1])
	if err != nil {
		log.Fatalf("error: CIDR address did not parse: %s", err)
	}

	u, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	scanResults := scanPrefixes(ipCh)
	unameResults := unamePrefixes(u.Username, scanResults)

	for rec := range unameResults {
		b, _ := json.Marshal(rec)
		fmt.Printf("%s\n", b)
	}
}

// record holds information about a scan of a host.
type record struct {
	// Host is the IP address of the host.
	Host net.IP
	// Reachable indicates if this host was pingable.
	Reachable bool
	// LoginSSH indicates if we were able to authenticate with SSH.
	LoginSSH bool
	// Uname is the output of the "uname -a" command. If this is an empty string
	// but LoginSSH is true, this means uname was not supported by the host.
	Uname string
}

// host takes a CIDR string (192.168.0.0/24) and returns all host IPs for that network.
// This will not send back the broadcast or network addresses. Does not support /31 addresses.
func hosts(cidr string) (chan net.IP, error) {
	ch := make(chan net.IP, 1)

	prefix, err := netaddr.ParseIPPrefix(cidr)
	if err != nil {
		return nil, err
	}

	go func() {
		defer close(ch)

		var last net.IP
		for ip := prefix.IP().Next(); prefix.Contains(ip); ip = ip.Next() {
			// Prevents sending the broadcast address.
			if len(last) != 0 {
				//log.Printf("sending: %s, contained: %v", last, prefix.Contains(ip))
				ch <- last
			}
			last = ip.IPAddr().IP
		}
	}()
	return ch, nil
}

// scanPrefixes takes a channel of net.IP and pings them. If an IP responds to ping it is put
// on the returned success channel, otherwise it is put on the fail channel.
func scanPrefixes(ipCh chan net.IP) chan record {
	ch := make(chan record, 1)
	go func() {
		defer close(ch)

		limit := make(chan struct{}, 100)
		wg := sync.WaitGroup{}
		for ip := range ipCh {
			limit <- struct{}{}
			wg.Add(1)
			go func(ip net.IP) {
				defer func() { <-limit }()
				defer wg.Done()

				ctx, cancel := context.WithTimeout(
					context.Background(),
					3*time.Second,
				)
				defer cancel()

				rec := record{Host: ip}
				if hostAlive(ctx, ip) {
					rec.Reachable = true
				}
				ch <- rec
			}(ip)
		}
		wg.Wait()
	}()
	return ch
}

// unamePrefixes takes a channel of net.IP and runs "uname -a" on them via the ssh binary.
func unamePrefixes(user string, recs chan record) chan record {
	ch := make(chan record, 1)
	go func() {
		defer close(ch)

		limit := make(chan struct{}, 100)
		wg := sync.WaitGroup{}
		for rec := range recs {
			if !rec.Reachable {
				ch <- rec
				continue
			}

			limit <- struct{}{}
			wg.Add(1)
			go func(rec record) {
				defer func() { <-limit }()
				defer wg.Done()

				text, err := runUname(context.Background(), rec.Host, user)
				if err != nil {
					ch <- rec
					return
				}
				rec.LoginSSH = true
				rec.Uname = text
				ch <- rec
			}(rec)
		}
		wg.Wait()
	}()
	return ch
}

// hostAlive uses the "ping" binary on the host to test an IP. Has a 2 second timeout.
func hostAlive(ctx context.Context, host net.IP) bool {
	cmd := exec.CommandContext(ctx, ping, "-c", "1", "-t", "2", host.String())

	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// runUname will attempt to use the "ssh" binary to log into a host and run "uname -a".
// This will return the output of that command.
func runUname(ctx context.Context, host net.IP, user string) (string, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}

	login := fmt.Sprintf("%s@%s", user, host)
	cmd := exec.CommandContext(
		ctx,
		ssh,
		"-o StrictHostKeyChecking=no",
		"-o BatchMode=yes",
		login,
		"uname -a",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
