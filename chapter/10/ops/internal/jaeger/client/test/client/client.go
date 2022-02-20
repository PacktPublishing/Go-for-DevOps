package client

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

// NestedSpans is the number of nested spans to create. Ths is set to 2560, because for some
// reason that is the maximum number of spans. That may be due to some byte limit or a fixed number.
// I could find no documentation to indicate why, thought I didn't look throughthe source.
const NestedSpans = 2560

// Initializes an OTLP exporter, and configures the corresponding trace providers.
func initProvider() func() {
	ctx := context.Background()

	traceExp := initTracer(ctx, "127.0.0.1:4317")
	log.Println("intTracer done")
	return func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := traceExp.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}
}

func initTracer(ctx context.Context, otelAgentAddr string) *otlptrace.Exporter {
	traceClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(otelAgentAddr),
		otlptracegrpc.WithDialOption(grpc.WithBlock(), grpc.WithTimeout(time.Second)))
	traceExp, err := otlptrace.New(ctx, traceClient)
	handleErr(err, "Failed to create the collector trace exporter")

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String("demo-client"),
		),
	)
	handleErr(err, "failed to create resource")

	bsp := sdktrace.NewBatchSpanProcessor(traceExp)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tracerProvider)
	return traceExp
}

func handleErr(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}

var once sync.Once

var Shutdown func() = func() {}

type HTTP struct {
	addr   string
	client http.Client
}

func New(addr string) (*HTTP, error) {
	once.Do(
		func() {
			log.Println("before initProvider")
			Shutdown = initProvider()
			log.Println("after initProvider")
		},
	)

	h := &HTTP{client: http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}}
	log.Println("after *HTTP client")
	return h, nil
}

func (h *HTTP) Call(ctx context.Context) (traceID string, err error) {
	tracer := otel.Tracer("demo-client-tracer")

	ctx, span := tracer.Start(ctx, "ExecuteRequest")
	defer span.End()

	ctx = h.makeNestedSpans(ctx, tracer)
	h.makeRequest(ctx)

	log.Println("trace says: ", span.SpanContext().TraceID().String())
	log.Println("convert says: ", convertTraceID(span.SpanContext().TraceID().String()))

	return span.SpanContext().TraceID().String(), nil
}

func (h *HTTP) makeRequest(ctx context.Context) error {
	// Make sure we pass the context to the request to avoid broken traces.
	req, err := http.NewRequestWithContext(ctx, "GET", h.addr, nil)
	if err != nil {
		return err
	}

	// All requests made with this client will create spans.
	res, err := h.client.Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()
	return nil
}

func (h *HTTP) makeNestedSpans(ctx context.Context, tracer trace.Tracer) context.Context {
	spans := []trace.Span{}
	for i := 0; i < NestedSpans; i++ {
		var span trace.Span
		ctx, span = tracer.Start(ctx, uuid.New().String())
		spans = append(spans, span)
	}
	for i := NestedSpans - 1; i > -1; i-- {
		spans[i].End()
	}
	return ctx
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
