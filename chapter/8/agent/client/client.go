/*
Package client provides a client to the system agent that uses SSH and unix sockets
to make the connection.

The SSH forwarding is based on code from:
https://stackoverflow.com/questions/21417223/simple-ssh-port-forward-in-golang
*/
package client

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/johnsiilver/serveonssh"
	"golang.org/x/crypto/ssh"
	"google.golang.org/grpc"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/8/agent/proto"
)

type Client struct {
	user     string
	endpoint string
	conn     *grpc.ClientConn
	client   pb.AgentClient
	p        serveonssh.Proxy
}

// New creates a new Client that connects to a remote endpoint via SSH and then
// uses that connection to dial into a domain socket the agent is using. The
// gRPC client actually uses a domain socket on this side which is then forwarded
// over SSH. endpoint is the host:port of the remote endpoint.
func New(endpoint string, auth []ssh.AuthMethod) (*Client, error) {
	config := &ssh.ClientConfig{
		User:            os.Getenv("USER"),
		Auth:            auth,
		Timeout:         5 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	remoteSocket := filepath.Join("/home", config.User, "/sa/socket/sa.sock")

	p, err := serveonssh.New(endpoint, remoteSocket, config)
	if err != nil {
		return nil, err
	}

	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return p.Dialer()()
		}),
	}

	conn, err := grpc.Dial("not needed", opts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		endpoint: endpoint,
		conn:     conn,
		client:   pb.NewAgentClient(conn),
		p:        p,
	}, nil
}

func (c *Client) Close() error {
	c.conn.Close()
	c.p.Close()
	return nil
}

func (c *Client) Install(ctx context.Context, req *pb.InstallReq) (*pb.InstallResp, error) {
	return c.client.Install(ctx, req)
}

func (c *Client) Remove(ctx context.Context, req *pb.RemoveReq) (*pb.RemoveResp, error) {
	return c.client.Remove(ctx, req)
}
