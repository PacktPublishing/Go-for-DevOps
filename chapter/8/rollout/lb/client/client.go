package client

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/8/rollout/lb/proto"
)

// HealthChecks are a set of backend health checks that a backend must pass
// in order to be considered healthy.
type HealthChecks struct {
	// HealthChecks are the checks to run against the backends.
	HealthChecks []HealthCheck
	// Interval is the between checks.
	Interval time.Duration
}

func (h HealthChecks) toPB() *pb.HealthChecks {
	hcs := &pb.HealthChecks{
		IntervalSecs: int32(h.Interval / time.Second),
	}
	for _, hc := range h.HealthChecks {
		p := hc.toPB()
		hcs.HealthChecks = append(hcs.HealthChecks, p)
	}
	return hcs
}

// HealthCheck defines a health check that must pass for a backend in a Pool
// to be considered healthy.
type HealthCheck interface {
	toPB() *pb.HealthCheck
	isHealthCheck()
}

// StatusCheck implements HealthCheck to check a node at "URLPath" for
// any string in "HealthyValues". If found, the node is said to be healthy.
type StatusCheck struct {
	// URLPath is the path to the health status page, like "/health".
	URLPath string
	// HealthValues are values that the URLPath can return and the service
	// is considered healthy.
	HealthyValues []string
}

func (s StatusCheck) toPB() *pb.HealthCheck {
	return &pb.HealthCheck{
		HealthCheck: &pb.HealthCheck_StatusCheck{
			StatusCheck: &pb.StatusCheck{
				UrlPath:       s.URLPath,
				HealthyValues: s.HealthyValues,
			},
		},
	}
}

func (s StatusCheck) isHealthCheck() {}

// Client is a client to the Quote of the day server.
type Client struct {
	client pb.LoadBalancerClient
	conn   *grpc.ClientConn
}

// New is the constructor for Client. addr is the server's [host]:[port].
func New(addr string) (*Client, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return &Client{
		client: pb.NewLoadBalancerClient(conn),
		conn:   conn,
	}, nil
}

// AddPool adds a pool that serves "pattern" using a PoolType that controls how
// the pool load balances traffic and a HealthCheck to determine if a node is healthy.
func (c *Client) AddPool(ctx context.Context, pattern string, pt pb.PoolType, hcs HealthChecks) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	_, err := c.client.AddPool(
		ctx,
		&pb.AddPoolReq{
			Pattern:      pattern,
			PoolType:     pt,
			HealthChecks: hcs.toPB(),
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// RemovePool removes the pool that is serving "pattern".
func (c *Client) RemovePool(ctx context.Context, pattern string) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	_, err := c.client.RemovePool(ctx, &pb.RemovePoolReq{Pattern: pattern})
	if err != nil {
		return err
	}
	return nil
}

// Backend represents a load balancer backend.
type Backend interface {
	isBackend()
}

// IPBackend implements Backend.
type IPBackend struct {
	IP      net.IP
	Port    int32
	URLPath string
}

func (i IPBackend) isBackend() {}

// AddBackend adds backend "b" from the pool serving "pattern".
func (c *Client) AddBackend(ctx context.Context, pattern string, b Backend) error {
	switch v := b.(type) {
	case IPBackend:
		return c.addIPBackend(ctx, pattern, v)
	}
	return fmt.Errorf("Backend is not a recognized type(%T)", b)
}

func (c *Client) addIPBackend(ctx context.Context, pattern string, b IPBackend) error {
	_, err := c.client.AddBackend(
		ctx,
		&pb.AddBackendReq{
			Pattern: pattern,
			Backend: &pb.Backend{
				Backend: &pb.Backend_IpBackend{
					IpBackend: &pb.IPBackend{
						Ip:      b.IP.String(),
						Port:    b.Port,
						UrlPath: b.URLPath,
					},
				},
			},
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// RemoveBackend removes backend "b" from the pool serving "pattern".
func (c *Client) RemoveBackend(ctx context.Context, pattern string, b Backend) error {
	switch v := b.(type) {
	case IPBackend:
		return c.removeIPBackend(ctx, pattern, v)
	}
	return fmt.Errorf("Backend is not a recognized type(%T)", b)
}

func (c *Client) removeIPBackend(ctx context.Context, pattern string, b IPBackend) error {
	_, err := c.client.RemoveBackend(
		ctx,
		&pb.RemoveBackendReq{
			Pattern: pattern,
			Backend: &pb.Backend{
				Backend: &pb.Backend_IpBackend{
					IpBackend: &pb.IPBackend{
						Ip:      b.IP.String(),
						Port:    b.Port,
						UrlPath: b.URLPath,
					},
				},
			},
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// PoolHealth queries the server for the health of the pool that serves "pattern".
// healthy and sick determine what node information is included.
func (c *Client) PoolHealth(ctx context.Context, pattern string, healthy, sick bool) (*pb.PoolHealth, error) {
	resp, err := c.client.PoolHealth(
		ctx,
		&pb.PoolHealthReq{
			Pattern: pattern,
			Healthy: healthy,
			Sick:    sick,
		},
	)
	if err != nil {
		return nil, err
	}
	return resp.Health, nil
}
