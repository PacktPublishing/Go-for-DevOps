// Package server contains our gRPC server implementation for the ops server.
package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sort"
	"sync"
	"time"

	jaeger "github.com/PacktPublishing/Go-for-DevOps/chapter/11/ops/internal/jaeger/client"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/11/ops/internal/prom"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/11/petstore/client"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/durationpb"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/11/ops/proto"
	mpb "github.com/PacktPublishing/Go-for-DevOps/chapter/11/ops/proto/jaeger/model"
)

// API implements our gRPC server's API.
type API struct {
	pb.UnimplementedOpsServer

	addr string

	grpcServer *grpc.Server
	gOpts      []grpc.ServerOption
	mu         sync.Mutex

	clients Clients
}

// Clients holds the remote clients requires to do ops.
type Clients struct {
	// Jaeger provides access to traces.
	Jaeger *jaeger.Jaeger
	// Prom provides access to metrics.
	Prom *prom.Client
	// Petstore provides access to the petstore.
	Petstore *client.Client
}

func (c Clients) validate() error {
	if c.Jaeger == nil {
		return errors.New("Jaeger cannot be nil")
	}
	if c.Prom == nil {
		return errors.New("Prom cannot be nil")
	}
	if c.Petstore == nil {
		return errors.New("PetStore cannot be nil")
	}
	return nil
}

// Option is an optional arguments to New().
type Option func(a *API)

// WithGRPCOpts creates the gRPC server with the options passed.
func WithGRPCOpts(opts ...grpc.ServerOption) Option {
	return func(a *API) {
		a.gOpts = append(a.gOpts, opts...)
	}
}

// New is the constructore for API.
func New(addr string, clients Clients, options ...Option) (*API, error) {
	if err := clients.validate(); err != nil {
		return nil, err
	}

	a := &API{addr: addr, clients: clients}

	for _, o := range options {
		o(a)
	}

	a.grpcServer = grpc.NewServer(a.gOpts...)
	a.grpcServer.RegisterService(&pb.Ops_ServiceDesc, a)
	reflection.Register(a.grpcServer)

	return a, nil
}

// Start starts the server. This blocks until Stop() is called.
func (a *API) Start() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	lis, err := net.Listen("tcp", a.addr)
	if err != nil {
		return err
	}

	return a.grpcServer.Serve(lis)
}

// Stop stops the server.
func (a *API) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.grpcServer.Stop()
}

// ListTraces lists recent traces from Jaeger for the petstore service.
func (a *API) ListTraces(ctx context.Context, req *pb.ListTracesReq) (*pb.ListTracesResp, error) {
	log.Println("tags:", req.Tags)
	params := jaeger.SearchParams{
		Service:     "petstore",
		Operation:   req.Operation,
		Tags:        req.Tags,
		DurationMin: time.Duration(req.DurationMin),
		DurationMax: time.Duration(req.DurationMax),
		SearchDepth: req.SearchDepth,
	}

	if req.Start > 0 {
		params.Start = time.Unix(0, req.Start)
	}
	if req.End > 0 {
		params.End = time.Unix(0, req.End)
	}

	ch, err := a.clients.Jaeger.Search(ctx, params)
	if err != nil {
		return nil, err
	}

	resp := &pb.ListTracesResp{}
	for trace := range ch {
		if len(trace.Spans) < 0 {
			continue
		}
		if trace.Err != nil {
			return nil, trace.Err
		}

		start := trace.Spans[0].Span.StartTime
		t := time.Unix(int64(start.Seconds), int64(start.Nanos)).UTC()
		resp.Traces = append(resp.Traces, &pb.TraceItem{Start: t.UnixNano(), Id: trace.ID})
	}
	sort.Slice(
		resp.Traces,
		func(i, j int) bool {
			if resp.Traces[i].Start > resp.Traces[j].Start {
				return true
			}
			return false
		},
	)
	return resp, nil
}

func (a *API) ShowLogs(ctx context.Context, req *pb.ShowLogsReq) (*pb.ShowLogsResp, error) {
	t, err := a.clients.Jaeger.Trace(ctx, req.Id)
	if err != nil {
		if err == jaeger.ErrNotFound {
			return &pb.ShowLogsResp{}, nil
		}
		return nil, err
	}

	logs := []*mpb.Log{}
	for _, span := range t.Spans {
		logs = append(logs, span.Logs...)
	}

	/*
		sort.SliceStable(
			logs,
			func(i, j int) bool {
				if logs[i].Timestamp.AsTime().Before(logs[j].Timestamp.AsTime()) {
					return true
				}
				return false
			},
		)
	*/

	return &pb.ShowLogsResp{
		Id:   t.ID,
		Logs: logs,
	}, nil
}

func (a *API) ShowTrace(ctx context.Context, req *pb.ShowTraceReq) (*pb.ShowTraceResp, error) {
	t, err := a.clients.Jaeger.Trace(ctx, req.Id)
	if err != nil {
		if err == jaeger.ErrNotFound {
			return &pb.ShowTraceResp{}, nil
		}
		return nil, err
	}
	var (
		ops    []string
		errors []string
		tags   []string
		dur    *durationpb.Duration
	)
	for _, span := range t.Spans {
		ops = append(ops, span.OperationName)
		for _, kv := range span.Tags {
			if kv.Key == "error" {
				errors = append(errors, kv.VStr)
			}
			tags = append(tags, kv.Key)
		}
		if span.Duration.AsDuration() > dur.AsDuration() {
			dur = span.Duration
		}
	}

	return &pb.ShowTraceResp{
		Id:         t.ID,
		Operations: ops,
		Errors:     errors,
		Tags:       tags,
		Duration:   dur,
	}, nil
}

// ChangeSampling changes the sampling type and rate for the Petstore.
func (a *API) ChangeSampling(ctx context.Context, req *pb.ChangeSamplingReq) (*pb.ChangeSamplingResp, error) {
	sc := client.Sampler{
		Type: client.SamplerType(req.Type),
		Rate: req.FloatValue,
	}

	if err := a.clients.Petstore.ChangeSampler(ctx, sc); err != nil {
		return nil, err
	}
	return &pb.ChangeSamplingResp{}, nil
}

// DeployedVersion returns the version of the Petstore that prometheus says is current.
func (a *API) DeployedVersion(ctx context.Context, req *pb.DeployedVersionReq) (*pb.DeployedVersionResp, error) {
	mv, _, err := a.clients.Prom.Metric(ctx, "deployedVersion")
	if err != nil {
		return nil, fmt.Errorf("problem getting metric: %w", err)
	}
	return &pb.DeployedVersionResp{Version: mv.String()}, nil
}

// Alerts grabs all currnetly firing alerts.
func (a *API) Alerts(ctx context.Context, req *pb.AlertsReq) (*pb.AlertsResp, error) {
	labels := map[string]string{}
	for _, l := range req.Labels {
		labels[l] = ""
	}
	filter := prom.AlertFilter{
		Labels:   labels,
		ActiveAt: time.Unix(0, req.ActiveAt),
		States:   req.States,
	}

	ch, err := a.clients.Prom.Alerts(ctx, filter)
	if err != nil {
		return nil, err
	}

	resp := &pb.AlertsResp{}
	for a := range ch {
		resp.Alerts = append(
			resp.Alerts,
			&pb.Alert{
				State:    string(a.State),
				Value:    a.Value,
				ActiveAt: a.ActiveAt.UnixNano(),
			},
		)
	}
	return resp, nil
}
