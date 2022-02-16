package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// main sets up the trace providers and starts a loop to continuously call the server
func main() {
	shutdown := initTraceProvider()
	defer shutdown()

	continuouslySendRequests()
}

// initTraceProvider initializes an OTLP exporter, and configures the corresponding trace provider.
func initTraceProvider() func() {
	ctx := context.Background()

	otelAgentAddr, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if !ok {
		otelAgentAddr = "0.0.0.0:4317"
	}

	closeTraces := initTracer(ctx, otelAgentAddr)

	return func() {
		doneCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		// pushes any last exports to the receiver
		closeTraces(doneCtx)
	}
}

// initTracer initializes an OTLP trace exporter and registers the trace provider with the global context
func initTracer(ctx context.Context, otelAgentAddr string) func(context.Context) {
	traceClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(otelAgentAddr),
		otlptracegrpc.WithDialOption(grpc.WithBlock()))
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

	return func(doneCtx context.Context) {
		if err := traceExp.Shutdown(doneCtx); err != nil {
			otel.Handle(err)
		}
	}
}

// handleErr provides a simple way to handle errors and messages
func handleErr(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}

// continuouslySendRequests continuously sends requests to the server sleeping for a second after each request.
func continuouslySendRequests() {
	tracer := otel.Tracer("demo-client-tracer")

	for {
		ctx, span := tracer.Start(context.Background(), "ExecuteRequest")
		makeRequest(ctx)
		SuccessfullyFinishedRequestEvent(span)
		span.End()
		time.Sleep(time.Duration(1) * time.Second)
	}
}

// makeRequest sends requests to the server using an OTEL HTTP transport which will instrument the requests with traces.
func makeRequest(ctx context.Context) {

	demoServerAddr, ok := os.LookupEnv("DEMO_SERVER_ENDPOINT")
	if !ok {
		demoServerAddr = "http://0.0.0.0:7080/hello"
	}

	// Trace an HTTP client by wrapping the transport
	client := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	// Make sure we pass the context to the request to avoid broken traces.
	req, err := http.NewRequestWithContext(ctx, "GET", demoServerAddr, nil)
	if err != nil {
		handleErr(err, "failed to http request")
	}

	// All requests made with this client will create spans.
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	res.Body.Close()
}

// SuccessfullyFinishedRequestEvent adds an event to the span which is analogous with a log statement, but is included
// in the trace structure and provides more context than a log statement.
func SuccessfullyFinishedRequestEvent(span trace.Span, opts ...trace.EventOption) {
	opts = append(opts, trace.WithAttributes(attribute.String("someKey", "someValue")))
	span.AddEvent("successfully finished request operation", opts...)
}

// WithCorrelation adds span and trace IDs to a zap logger to enable better correlation between traces and logs.
func WithCorrelation(span trace.Span, log *zap.Logger) *zap.Logger {
	return log.With(
		zap.String("span_id", convertTraceID(span.SpanContext().SpanID().String())),
		zap.String("trace_id", convertTraceID(span.SpanContext().TraceID().String())),
	)
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
