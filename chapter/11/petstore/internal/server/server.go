// Package server contains our gRPC server implementation for the pet store.
package server

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/11/petstore/internal/server/errors"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/11/petstore/internal/server/storage"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/11/petstore/internal/server/telemetry/metrics"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/11/petstore/internal/server/telemetry/tracing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	otelCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/11/petstore/proto"
)

// These represent all of our OTEL metric counters.
var (
	totalCount, addCount, deleteCount, updateCount, searchCount metric.Int64Counter

	addCurrent, deleteCurrent, updateCurrent, searchCurrent metric.Int64UpDownCounter

	addLat, deleteLat, updateLat, searchLat metric.Int64Histogram

	addErrors, deleteErrors, updateErrors, searchErrors metric.Int64Counter
)

// This fetches all of our counters. You can only do this in init().
func init() {
	totalCount = metrics.Get.Int64("petstore/server/totals/requests")
	addCount = metrics.Get.Int64("petstore/server/AddPets/requests")
	deleteCount = metrics.Get.Int64("petstore/server/DeletePets/requests")
	updateCount = metrics.Get.Int64("petstore/server/UpdatePets/requests")
	searchCount = metrics.Get.Int64("petstore/server/SearchPets/requests")

	addCurrent = metrics.Get.Int64UD("petstore/server/AddPets/current")
	deleteCurrent = metrics.Get.Int64UD("petstore/server/DeletePets/current")
	updateCurrent = metrics.Get.Int64UD("petstore/server/UpdatePets/current")
	searchCurrent = metrics.Get.Int64UD("petstore/server/SearchPets/current")

	addErrors = metrics.Get.Int64("petstore/server/AddPets/errors")
	deleteErrors = metrics.Get.Int64("petstore/server/DeletePets/errors")
	updateErrors = metrics.Get.Int64("petstore/server/UpdatePets/errors")
	searchErrors = metrics.Get.Int64("petstore/server/SearchPets/errors")

	addLat = metrics.Get.Int64Hist("petstore/server/AddPets/latency")
	deleteLat = metrics.Get.Int64Hist("petstore/server/DeletePets/latency")
	updateLat = metrics.Get.Int64Hist("petstore/server/UpdatePets/latency")
	searchLat = metrics.Get.Int64Hist("petstore/server/SearchPets/latency")
}

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

// AddPets adds pets to the pet store.
func (a *API) AddPets(ctx context.Context, req *pb.AddPetsReq) (resp *pb.AddPetsResp, err error) {
	// Handle tracing.
	ctx, _, end := doTrace(ctx, "server.AddPets()", req)
	defer func() { end(err) }()

	// Handle metrics.
	metrics.Meter.RecordBatch(
		ctx,
		nil,
		totalCount.Measurement(1),
		addCount.Measurement(1),
		addCurrent.Measurement(1),
	)
	t := time.Now()
	defer func() {
		metrics.Meter.RecordBatch(
			ctx,
			nil,
			addCurrent.Measurement(-1),
			addLat.Measurement(int64(time.Since(t))),
		)
		if err != nil {
			code := status.Code(err)
			addErrors.Add(ctx, 1, attribute.String("code", code.String()))
		}
	}()

	// Actual work.
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

	// Handle metrics.
	metrics.Meter.RecordBatch(
		ctx,
		nil,
		totalCount.Measurement(1),
		updateCount.Measurement(1),
		updateCurrent.Measurement(1),
	)
	t := time.Now()
	defer func() {
		metrics.Meter.RecordBatch(
			ctx,
			nil,
			updateCurrent.Measurement(-1),
			updateLat.Measurement(int64(time.Since(t))),
		)
		if err != nil {
			code := status.Code(err)
			updateErrors.Add(ctx, 1, attribute.String("code", code.String()))
		}
	}()

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

	// Handle metrics.
	metrics.Meter.RecordBatch(
		ctx,
		nil,
		totalCount.Measurement(1),
		deleteCount.Measurement(1),
		deleteCurrent.Measurement(1),
	)
	t := time.Now()
	defer func() {
		metrics.Meter.RecordBatch(
			ctx,
			nil,
			deleteCurrent.Measurement(-1),
			deleteLat.Measurement(int64(time.Since(t))),
		)
		if err != nil {
			code := status.Code(err)
			deleteErrors.Add(ctx, 1, attribute.String("code", code.String()))
		}
	}()

	if err = a.store.DeletePets(ctx, req.Ids); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.DeletePetsResp{}, nil
}

// SearchPets finds pets in the pet store.
func (a *API) SearchPets(req *pb.SearchPetsReq, stream pb.PetStore_SearchPetsServer) (err error) {
	count := 0

	ctx, span, end := doTrace(stream.Context(), "server.SearchPets()", req)
	defer func() { end(err) }()
	defer func() {
		span.SetAttributes(attribute.Int("search.results.returned", count))
	}()

	// Handle metrics.
	metrics.Meter.RecordBatch(
		ctx,
		nil,
		totalCount.Measurement(1),
		searchCount.Measurement(1),
		searchCurrent.Measurement(1),
	)
	t := time.Now()
	defer func() {
		metrics.Meter.RecordBatch(
			ctx,
			nil,
			searchCurrent.Measurement(-1),
			searchLat.Measurement(int64(time.Since(t))),
		)
		if err != nil {
			code := status.Code(err)
			searchErrors.Add(ctx, 1, attribute.String("code", code.String()))
		}
	}()

	if err = validateSearch(ctx, req); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	ch := a.store.SearchPets(ctx, req)
	for item := range ch {
		count++
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
	switch req.Sampler.Type {
	case pb.SamplerType_STUnknown:
		// Skip, will return an error
	case pb.SamplerType_STNever:
		tracing.Sampler.Switch(sdkTrace.NeverSample())
		return &pb.ChangeSamplerResp{}, nil
	case pb.SamplerType_STAlways:
		tracing.Sampler.Switch(sdkTrace.AlwaysSample())
		return &pb.ChangeSamplerResp{}, nil
	case pb.SamplerType_STFloat:
		if req.Sampler.FloatValue <= 0 || req.Sampler.FloatValue > 1 {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("float_value=%v is invalid", req.Sampler.FloatValue))
		}
		tracing.Sampler.Switch(sdkTrace.TraceIDRatioBased(req.Sampler.FloatValue))
		return &pb.ChangeSamplerResp{}, nil
	}
	return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("type==%v is invalid", req.Sampler.Type))
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
	ctx, span = tracing.Tracer.Start(
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

	// If they asked for a trace, send back the trace ID.
	if ctx.Value("trace") != nil {
		id := span.SpanContext().TraceID().String()
		if id != "" {
			header := metadata.Pairs("traceID", convertTraceID(id))
			grpc.SendHeader(ctx, header)
		}
	}

	return ctx, span, func(err error) {
		if err != nil {
			span.SetStatus(otelCodes.Error, err.Error())
			span.SetAttributes(
				attribute.Bool("error", true),
				attribute.String("errorMsg", err.Error()),
			)
			span.End()
			return
		}
		span.SetStatus(otelCodes.Ok, "")
		span.End()
	}
}

func convertTraceID(id string) string {
	if len(id) < 16 {
		return ""
	}
	if len(id) > 16 {
		id = id[16:]
	}
	intValue, err := strconv.ParseUint(id, 16, 64)
	if err != nil {
		return ""
	}
	return strconv.FormatUint(intValue, 10)
}
