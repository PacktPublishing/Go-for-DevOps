// Package client provides a Jaegar client for grabbing traces from the Jaegar service.
// It is a wrapper around the undocumented Jaegar gRPC client.
package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	otelTrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	duration "google.golang.org/protobuf/types/known/durationpb"
	timestamp "google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/11/ops/proto/jaeger"
	mpb "github.com/PacktPublishing/Go-for-DevOps/chapter/11/ops/proto/jaeger/model"
)

var (
	// ErrNotFound indicates that no trace with an ID was found.
	ErrNotFound = errors.New("trace with that ID was not found")
)

// Trace represents a OTEL trace that was stored in Jaegar.
type Trace struct {
	// ID is the identity of the trace.
	ID string
	// Spans are the spans that make up the trace.
	Spans []Span

	// Err indicates if there was a error in the trace stream if the
	// Trace is being returned in a channel. If not, this will always be nil.
	Err error
}

// Span is a convienence wrapper around *mpb.Span.
type Span struct {
	*mpb.Span
}

// Proto returns the encapsulated proto. Remember that these are generated with
// gogo proto and not Google/Buf.build proto engine.
func (s Span) Proto() *mpb.Span {
	return s.Span
}

// TraceID returns the converted human readable Trace ID.
func (s Span) TraceID() string {
	if len(s.Span.TraceId) < 16 {
		return ""
	}
	// This is a go 1.17 conversion of a slice to an array.
	t := (*otelTrace.TraceID)(s.Span.TraceId[0:16])
	return t.String()
}

// SpanID returns the converted human readable Span ID.
func (s Span) SpanID() string {
	if len(s.Span.SpanId) < 16 {
		return ""
	}
	t := (*otelTrace.SpanID)(s.Span.SpanId[0:16])
	return t.String()
}

// Jaeger provides a client for interacting with Jaeger to retrieve traces.
type Jaeger struct {
	client pb.QueryServiceClient
	conn   *grpc.ClientConn
	addr   string
}

// New creates a new Jaeger client that connects to addr.
func New(addr string) (*Jaeger, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return &Jaeger{
		client: pb.NewQueryServiceClient(conn),
		conn:   conn,
		addr:   addr,
	}, nil
}

// Addr returns the address of the Jaeger server we are connected to.
func (j *Jaeger) Addr() string {
	return j.addr
}

// SearchParams are parameters used to filter a search for trace data.
type SearchParams struct {
	// Service is the name of the service you are querying. Required.
	Service string
	// Operation is the name of the operation you want to filter by.
	Operation string
	// Tag values you want to filter by.
	Tags []string
	// Start is the lower bounds (inclusive) that a trace can start at.
	Start time.Time
	// End is the uppper bounds (exclusive) a trace can end at.
	End time.Time
	// DurationMin is the minimum duration a trace (inclusive) must have.
	DurationMin time.Duration
	// DurationMax is the maximum duration a trace (exclusive) can have.
	DurationMax time.Duration
	// SearchDepth is a quirky setting. It is kinda tells the data store how hard to search.
	// On data stores like Cassandra, settting this to a higher number will cause it
	// to search deeper in its trees for the data. So a setting of 20 might get 8 results, but 200 will get
	// 15. This looks to have only a limit like effect on other storage systems. This defaults to 200.
	SearchDepth int32
}

func (s SearchParams) validate() error {
	if s.Service == "" {
		return errors.New("Service field must not be an empty string")
	}
	return nil
}

func (s SearchParams) proto() *pb.FindTracesRequest {
	if s.SearchDepth == 0 {
		s.SearchDepth = 20
	}
	var t map[string]string
	if len(s.Tags) > 0 {
		t = make(map[string]string, len(s.Tags))
		for _, tag := range s.Tags {
			t[tag] = ""
		}
	}

	return &pb.FindTracesRequest{
		Query: &pb.TraceQueryParameters{
			ServiceName:   s.Service,
			OperationName: s.Operation,
			Tags:          t,
			StartTimeMin:  timestamp.New(s.Start),
			StartTimeMax:  timestamp.New(s.End),
			DurationMin:   duration.New(s.DurationMin),
			DurationMax:   duration.New(s.DurationMax),
			SearchDepth:   s.SearchDepth,
		},
	}
}

// Search searches Jaeger for traces that match the SearchParams. Each result is the set of spans that make up a trace.
func (j *Jaeger) Search(ctx context.Context, params SearchParams) (chan Trace, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}

	stream, err := j.client.FindTraces(ctx, params.proto())
	if err != nil {
		return nil, err
	}

	// Traces come in chunks of spans. So we look at the chunks that come in and combine spans with the same IDs into traces.
	return unwind(ctx, stream), nil
}

// Trace allows retreival of a specific trace from Jaegar by its ID.
func (j *Jaeger) Trace(ctx context.Context, id string) (Trace, error) {
	tid, err := otelTrace.TraceIDFromHex(id)
	if err != nil {
		return Trace{}, fmt.Errorf("trace ID was invalid: %w", err)
	}

	req := &pb.GetTraceRequest{
		TraceId: tid[0:16],
	}

	stream, err := j.client.GetTrace(ctx, req)
	if err != nil {
		return Trace{}, err
	}

	ch := unwind(ctx, stream)
	traces := make([]Trace, 0, 1)
	for trace := range ch {
		traces = append(traces, trace)
	}
	switch len(traces) {
	case 0:
		return Trace{}, ErrNotFound
	case 1:
		return traces[0], nil
	}
	return Trace{}, fmt.Errorf("bug: received more that a single Trace")
}

type receiver interface {
	Recv() (*pb.SpansResponseChunk, error)
	grpc.ClientStream
}

// unwind unwinds traces that come in chunks of spans. So we look at the chunks that come in and combine spans with the same IDs into traces.
func unwind(ctx context.Context, stream receiver) chan Trace {
	ch := make(chan Trace, 1)
	go func() {
		defer close(ch)
		var lastTrace Trace
		for {
			if ctx.Err() != nil {
				ch <- Trace{Err: ctx.Err()}
				return
			}
			chunk, err := stream.Recv()
			if err == io.EOF {
				if lastTrace.ID != "" {
					ch <- lastTrace
				}
				return
			}
			if err != nil {
				ch <- Trace{Err: err}
				return
			}
			spans := chunkToSpan(chunk)
			if len(spans) == 0 {
				continue
			}
			if spans[0].TraceID() != lastTrace.ID {
				if lastTrace.ID != "" {
					ch <- lastTrace
				}
				lastTrace = Trace{ID: spans[0].TraceID(), Spans: spans}
			} else {
				lastTrace.Spans = append(lastTrace.Spans, spans...)
			}
		}
	}()
	return ch
}

func chunkToSpan(chunk *pb.SpansResponseChunk) []Span {
	if len(chunk.Spans) == 0 {
		return nil
	}
	var spans []Span
	for _, s := range chunk.Spans {
		spans = append(spans, Span{Span: s})
	}
	return spans
}
