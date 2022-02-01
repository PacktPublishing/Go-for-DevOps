// Package server contains our gRPC server implementation for the pet store.
package server

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/petstore/internal/server/errors"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/petstore/internal/server/storage"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/petstore/internal/server/telemetry"

	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/8/petstore/proto"
)

// API implements our gRPC server's API.
type API struct {
	pb.UnimplementedPetStoreServer

	addr  string
	store storage.Data

	grpcServer *grpc.Server
	gOpts      []grpc.ServerOption
	mu         sync.Mutex
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
func New(addr string, store storage.Data, options ...Option) (*API, error) {
	a := &API{addr: addr, store: store}

	for _, o := range options {
		o(a)
	}

	a.grpcServer = grpc.NewServer(a.gOpts...)
	a.grpcServer.RegisterService(&pb.PetStore_ServiceDesc, a)

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

// AddPets adds pets to the pet store.
func (a *API) AddPets(ctx context.Context, req *pb.AddPetsReq) (resp *pb.AddPetsResp, err error) {
	ctx, _, end := doTrace(ctx, "server.AddPets()", req)
	defer func() { end(err) }()

	ids := make([]string, 0, len(req.Pets))
	for _, p := range req.Pets {
		if err := storage.ValidatePet(ctx, p, false); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		p.Id = uuid.New().String()
		ids = append(ids, p.Id)
	}

	if err = a.store.AddPets(ctx, req.Pets); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.AddPetsResp{Ids: ids}, nil
}

// UpdatePets updates pets in the pet store.
func (a *API) UpdatePets(ctx context.Context, req *pb.UpdatePetsReq) (resp *pb.UpdatePetsResp, err error) {
	ctx, _, end := doTrace(ctx, "server.UpdatePets()", req)
	defer func() { end(err) }()

	for _, p := range req.Pets {
		if err = storage.ValidatePet(ctx, p, true); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	if err = a.store.UpdatePets(ctx, req.Pets); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.UpdatePetsResp{}, nil
}

// DeletePets deletes pets from the pet store.
func (a *API) DeletePets(ctx context.Context, req *pb.DeletePetsReq) (resp *pb.DeletePetsResp, err error) {
	ctx, _, end := doTrace(ctx, "server.DeletePets()", req)
	defer func() { end(err) }()

	if err = a.store.DeletePets(ctx, req.Ids); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.DeletePetsResp{}, nil
}

// SearchPets finds pets in the pet store.
func (a *API) SearchPets(req *pb.SearchPetsReq, stream pb.PetStore_SearchPetsServer) (err error) {
	ctx, _, end := doTrace(stream.Context(), "server.SearchPets()", req)
	defer func() { end(err) }()

	if err = validateSearch(ctx, req); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	ch := a.store.SearchPets(ctx, req)
	for item := range ch {
		if item.Error != nil {
			return status.Error(codes.Internal, item.Error.Error())
		}
		if err := stream.Send(item.Pet); err != nil {
			return err
		}
	}
	if ctx.Err() != nil {
		return status.Error(codes.DeadlineExceeded, stream.Context().Err().Error())
	}
	return nil
}

// ChangeSampler changes the OTEL sampling type.
func (a *API) ChangeSampler(ctx context.Context, req *pb.ChangeSamplerReq) (resp *pb.ChangeSamplerResp, err error) {
	switch req.Type {
	case pb.SamplerType_STUnknown:
		// Skip, will return an error
	case pb.SamplerType_STNever:
		telemetry.Sampler.Switch(sdkTrace.NeverSample())
		return &pb.ChangeSamplerResp{}, nil
	case pb.SamplerType_STAlways:
		telemetry.Sampler.Switch(sdkTrace.AlwaysSample())
		return &pb.ChangeSamplerResp{}, nil
	case pb.SamplerType_STFloat:
		if req.FloatValue <= 0 || req.FloatValue > 1 {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("float_value=%v is invalid", req.FloatValue))
		}
		telemetry.Sampler.Switch(sdkTrace.TraceIDRatioBased(req.FloatValue))
		return &pb.ChangeSamplerResp{}, nil
	}
	return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("type==%v is invalid", req.Type))
}

func validateSearch(ctx context.Context, r *pb.SearchPetsReq) error {
	for _, t := range r.Types {
		if t == pb.PetType_PTUnknown {
			return errors.New(ctx, "cannot search for PetType_Unkonwn")
		}
	}

	if r.BirthdateRange != nil {
		if r.BirthdateRange.Start == nil {
			return errors.New(ctx, "cannot have a BirthdateRange.Start that is nil")
		}
		if r.BirthdateRange.End == nil {
			return errors.New(ctx, "cannot have a BirthdateRange.End that is nil")
		}
		if _, err := storage.BirthdayToTime(ctx, r.BirthdateRange.Start); err != nil {
			return errors.Errorf(ctx, "r.BirthdateRange.Start had error: %s", err)
		}
		if _, err := storage.BirthdayToTime(ctx, r.BirthdateRange.End); err != nil {
			return errors.Errorf(ctx, "r.BirthdateRange.End had error: %s", err)
		}
	}
	return nil
}

func doTrace(ctx context.Context, name string, req proto.Message) (newCtx context.Context, span trace.Span, end func(err error)) {
	ctx, span = telemetry.Tracer.Start(
		ctx,
		name,
		trace.WithAttributes(
			attribute.String("args", protojson.Format(req)),
			attribute.Bool("grpcCall", true),
		),
	)
	p, ok := peer.FromContext(ctx)
	if ok {
		host, port, err := net.SplitHostPort(p.Addr.String())
		if err == nil {
			portNum, _ := strconv.Atoi(port)
			span.SetAttributes(
				attribute.String("net.peer.ip", host),
				attribute.Int("net.peer.port", portNum),
			)
		}
	}

	return ctx, span, func(err error) {
		if err != nil {
			span.SetAttributes(attribute.String("rpcError", err.Error()))
		}
		span.End()
	}
}
