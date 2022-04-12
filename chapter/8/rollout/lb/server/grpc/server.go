// Package grpc implements a gRPC server for controlling our HTTP load balancer.
package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/rollout/lb/server/http"

	"google.golang.org/grpc"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/8/rollout/lb/proto"
)

// Server is a gRPC server for interacting with the load balancer.
type Server struct {
	pb.UnimplementedLoadBalancerServer

	addr       string
	lb         *http.LoadBalancer
	grpcServer *grpc.Server

	mu sync.Mutex
}

// New creates a new instance of Server.
func New(addr string, lb *http.LoadBalancer) (*Server, error) {
	var opts []grpc.ServerOption

	s := &Server{
		addr:       addr,
		lb:         lb,
		grpcServer: grpc.NewServer(opts...),
	}
	s.grpcServer.RegisterService(&pb.LoadBalancer_ServiceDesc, s)

	return s, nil
}

// Start starts the server and blocks.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	return s.grpcServer.Serve(lis)
}

// Stop stops the server.
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.grpcServer.Stop()
}

// AddPool adds a pool as defined in req.
func (s *Server) AddPool(ctx context.Context, req *pb.AddPoolReq) (*pb.AddPoolResp, error) {
	log.Println("adding pool")
	if strings.TrimSpace(req.Pattern) == "" {
		return nil, fmt.Errorf("pattern must not be empty")
	}

	if req.PoolType == pb.PoolType_PT_UNKNOWN {
		return nil, fmt.Errorf("must set a pool_type")
	}

	if len(req.HealthChecks.HealthChecks) == 0 {
		return nil, fmt.Errorf("must have at least 1 health_check")
	}

	var hcs []http.HealthCheck
	for _, hc := range req.HealthChecks.HealthChecks {
		switch {
		case hc.GetStatusCheck() != nil:
			scr := hc.GetStatusCheck()
			sc, err := http.StatusCheck(scr.UrlPath, scr.HealthyValues)
			if err != nil {
				return nil, err
			}
			hcs = append(hcs, sc)
		default:
			return nil, fmt.Errorf("a health_check is missing its concrete type")
		}
	}
	interval := time.Duration(req.HealthChecks.IntervalSecs) * time.Second

	var pool http.Pool

	switch req.PoolType {
	case pb.PoolType_PT_P2C:
		var err error
		pool, err = http.NewP2C(
			http.HealthMultiplexer(hcs...),
			interval,
		)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown pool_type(%v", req.PoolType)
	}

	if err := s.lb.AddPool(req.Pattern, pool); err != nil {
		return nil, err
	}
	return &pb.AddPoolResp{}, nil
}

// RemovePool removes a pool as defined in req.
func (s *Server) RemovePool(ctx context.Context, req *pb.RemovePoolReq) (*pb.RemovePoolResp, error) {
	if strings.TrimSpace(req.Pattern) == "" {
		return nil, fmt.Errorf("pattern must not be empty")
	}
	if err := s.lb.RemovePool(req.Pattern); err != nil {
		return nil, err
	}
	return &pb.RemovePoolResp{}, nil
}

// AddBackend adds a backend as defined in req.
func (s *Server) AddBackend(ctx context.Context, req *pb.AddBackendReq) (*pb.AddBackendResp, error) {
	log.Println("adding backend")
	if strings.TrimSpace(req.Pattern) == "" {
		return nil, fmt.Errorf("pattern must not be empty")
	}
	pool, err := s.lb.GetPool(req.Pattern)
	if err != nil {
		return nil, err
	}

	var back http.Backend

	switch {
	case req.Backend.GetIpBackend() != nil:
		v := req.Backend.GetIpBackend()
		ip := net.ParseIP(v.Ip)
		if ip == nil {
			return nil, fmt.Errorf("backend ip is invalid")
		}
		if v.Port < 1 || v.Port > 65534 {
			return nil, fmt.Errorf("port is invalid")
		}
		b, err := http.NewIPBackend(ip, v.Port, v.UrlPath)
		if err != nil {
			return nil, err
		}
		back = b
	default:
		return nil, fmt.Errorf("a backend is missing its concrete type")
	}
	if err := pool.Add(ctx, back); err != nil {
		return nil, err
	}
	return &pb.AddBackendResp{}, nil
}

// RemoveBackend remoes a backend as defined in req.
func (s *Server) RemoveBackend(ctx context.Context, req *pb.RemoveBackendReq) (*pb.RemoveBackendResp, error) {
	if strings.TrimSpace(req.Pattern) == "" {
		return nil, fmt.Errorf("pattern must not be empty")
	}

	var back http.Backend

	switch {
	case req.Backend.GetIpBackend() != nil:
		v := req.Backend.GetIpBackend()
		ip := net.ParseIP(v.Ip)
		if ip == nil {
			return nil, fmt.Errorf("backend ip is invalid")
		}
		if v.Port < 1 || v.Port > 65534 {
			return nil, fmt.Errorf("port is invalid")
		}
		b, err := http.NewIPBackend(ip, v.Port, v.UrlPath)
		if err != nil {
			return nil, err
		}
		back = b
	default:
		return nil, fmt.Errorf("a backend is missing its concrete type")
	}

	pool, err := s.lb.GetPool(req.Pattern)
	if err != nil {
		return nil, err
	}

	if err := pool.Remove(ctx, back); err != nil {
		return nil, err
	}
	return &pb.RemoveBackendResp{}, nil
}

// PoolHealth returns the health of a pool defined in req.
func (s *Server) PoolHealth(ctx context.Context, req *pb.PoolHealthReq) (*pb.PoolHealthResp, error) {
	ph, err := s.lb.PoolHealth(ctx, req)
	if err != nil {
		return nil, err
	}

	return &pb.PoolHealthResp{Health: ph}, nil
}
