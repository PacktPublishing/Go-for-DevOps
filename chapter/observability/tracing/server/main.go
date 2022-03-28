package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"os"
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
	"google.golang.org/grpc"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// main initializes tracing provider and listens to requests at /hello returning "Hello World!" with
// randomized latency.
func main() {
	shutdown := initTraceProvider()
	defer shutdown()

	// create a handler wrapped in OpenTelemetry instrumentation
	handler := handleRequestWithRandomSleep()
	wrappedHandler := otelhttp.NewHandler(handler, "/hello")

	// serve up the wrapped handler
	http.Handle("/hello", wrappedHandler)
	http.ListenAndServe(":7080", nil)
}

// handleRequestWithRandomSleep registers a request handler that will randomly sleep to induce artificial request latency.
func handleRequestWithRandomSleep() http.HandlerFunc {
	commonLabels := []attribute.KeyValue{
		attribute.String("server-attribute", "foo"),
	}

	return func(w http.ResponseWriter, req *http.Request) {
		//  random sleep to simulate latency
		var sleep int64
		switch modulus := time.Now().Unix() % 5; modulus {
		case 0:
			sleep = rng.Int63n(2000)
		case 1:
			sleep = rng.Int63n(15)
		case 2:
			sleep = rng.Int63n(917)
		case 3:
			sleep = rng.Int63n(87)
		case 4:
			sleep = rng.Int63n(1173)
		}
		time.Sleep(time.Duration(sleep) * time.Millisecond)
		ctx := req.Context()
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(commonLabels...)
		w.Write([]byte("Hello World"))
	}
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
			semconv.ServiceNameKey.String("demo-server"),
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

func handleErr(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}
