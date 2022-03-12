/*
Package tracing provides functions for starting and stopping our Open Telemetry tracing.
This package is intended to be used from main and is simple to use. We offer a few
choices on where traces export to. Here is an example to trace to stderr for all requests:
	func main() {
		ctx := context.Background()
		// Set us up to always sample. The "trace" package is: "petstore/server/SearchPets/latency"
		tracing.Sampler.Switch(trace.AlwaysSample())
		// Start our tracing and pass the empty Stderr tracing arguments.
		// Stderr{} has no required fields.
		stop, err := tracing.Start(ctx, tracing.Stderr{})
		if err != nil {
			log.Fatalf("problem starting telemetry: %s", err)
		}

		// Stop kills our exporter when main() ends.
		defer stop()
	}
*/
package tracing

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/14/petstore-operator/client/internal/server/telemetry/tracing/sampler"
)

// Tracer is the tracer initialized by Start().
var (
	// Tracer is the tracer initialized by Start().
	Tracer trace.Tracer // *sdktrace.TracerProvider //otlptrace.Exporter
	// Sampler is our *sampler.Sampler used by the Tracer.
	Sampler *sampler.Sampler
)

func init() {
	s, err := sampler.New(sdktrace.TraceIDRatioBased(1))
	if err != nil {
		panic(err)
	}
	Sampler = s
}

// Exporter represents the exporter to send telemetry to.
type Exporter interface {
	isExporter()
}

// OTELGRPC represents exporting to the go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc exporter.
type OTELGRPC struct {
	// Addr is the local address to export on.
	Addr string
}

func (o OTELGRPC) isExporter() {}

// Stderr exports trace data to os.Stderr.
type Stderr struct{}

func (s Stderr) isExporter() {}

// File exports trace data to a file. If the file exists, it is overwritten.
type File struct {
	// Path is the path to the file.
	Path string
}

func (f File) isExporter() {}

// Stop stops our Open Telemetry exporter.
type Stop func()

// Start creates the OTEL exporter and configures the trace providers.
// It returns a Stop() which will stop the exporter.
func Start(ctx context.Context, e Exporter) (Stop, error) {
	log.Println("Sampler: ", Sampler)
	tp, err := newTraceExporter(ctx, e)
	if err != nil {
		return nil, err
	}
	Tracer = tp.Tracer("petstore")

	return func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 1*time.Second)
		defer cancel()

		if err := tp.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}, nil
}

// newTracerExporter creates an OTLP exporter with our tracer information.
func newTraceExporter(ctx context.Context, e Exporter) (*sdktrace.TracerProvider, error) {
	var exp sdktrace.SpanExporter
	var err error
	switch v := e.(type) {
	case OTELGRPC:
		exp, err = otelGRPC(ctx, v)
	case Stderr:
		exp, err = newFileExporter(os.Stderr)
	case File:
		f, err := os.Create(v.Path)
		if err != nil {
			return nil, err
		}
		exp, err = newFileExporter(f)
	default:
		return nil, fmt.Errorf("%T is not a valid Exporter", e)
	}

	res, err := resource.New(
		ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String("petstore"),
		),
	)
	if err != nil {
		return nil, err
	}

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	prov := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(Sampler),
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(prov)
	return prov, nil
}

// newFileExporter creates an exporter that writes to a file.
func newFileExporter(w io.Writer) (sdktrace.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithWriter(w),
		stdouttrace.WithPrettyPrint(),
	)
}

func otelGRPC(ctx context.Context, e OTELGRPC) (sdktrace.SpanExporter, error) { //(*otlptrace.Exporter, error) {
	exp, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(e.Addr),
			otlptracegrpc.WithDialOption(grpc.WithBlock()),
		),
	)
	if err != nil {
		return nil, err
	}

	return exp, nil
}
