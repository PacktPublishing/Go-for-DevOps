/*
Package telemetry provides functions for starting and stopping our Open Telemetry tracing.
This package is intended to be used from main and is simple to use:
	var otelAddr = flag.String("otelAddr", "", "The address for our OpenTelemetry agent. If not set, looks for Env variable 'OTEL_EXPORTER_OTLP_ENDPOINT'. If not set defaults to 0.0.0:4317")

	func init() {
		if *otelAddr == "" {
			addr, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
			if !ok {
				addr = "0.0.0.0:4317"
			}
			*otelAddr = addr
		}
	}

	func main() {
		ctx := context.Background()
		stop, err := telemetry.Start(ctx, *otelAddr)
		if err != nil {
			log.Fatalf("problem starting telemetry: %s", err)
		}
		defer stop()
	}
*/
package telemetry

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/petstore/internal/server/telemetry/sampler"
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
)

// Tracer is the tracer initialized by Start().
var (
	// Tracer is the tracer initialized by Start().
	Tracer trace.Tracer
	// Sampler is our *sampler.Sampler used by the Tracer.
	Sampler *sampler.Sampler
)

func init() {
	s, err := sampler.New(sdktrace.TraceIDRatioBased(.01))
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
	tracer, err := newTraceExporter(ctx, e)
	if err != nil {
		return nil, err
	}

	return func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 1*time.Second)
		defer cancel()

		if err := tracer.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}, nil
}

// newFileExporter creates an exporter that writes to a file.
func newFileExporter(w io.Writer) (sdktrace.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithWriter(w),
		stdouttrace.WithPrettyPrint(),
	)
}

// newTracerExporter creates an OTLP exporter with our tracer information.
func newTraceExporter(ctx context.Context, e Exporter) (sdktrace.SpanExporter, error) {
	switch v := e.(type) {
	case OTELGRPC:
		return otelGRPC(ctx, v)
	case Stderr:
		return newFileExporter(os.Stderr)
	case File:
		f, err := os.Create(v.Path)
		if err != nil {
			return nil, err
		}
		return newFileExporter(f)
	default:
		return nil, fmt.Errorf("%T is not a valid Exporter", e)
	}
}

func otelGRPC(ctx context.Context, e OTELGRPC) (*otlptrace.Exporter, error) {
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
	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSampler(Sampler),
			sdktrace.WithResource(res),
			sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(exp)),
		),
	)
	return exp, nil
}
