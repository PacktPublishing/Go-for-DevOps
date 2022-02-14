package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc"
)

// main sets up the trace and metrics providers and starts a loop to continuously call the server
func main() {
	shutdown := initTraceAndMetricsProvider()
	defer shutdown()

	continuouslySendRequests()
}

// initTraceAndMetricsProvider initializes an OTLP exporter, and configures the corresponding trace and
// metric providers.
func initTraceAndMetricsProvider() func() {
	ctx := context.Background()

	otelAgentAddr, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if !ok {
		otelAgentAddr = "0.0.0.0:4317"
	}

	closeMetrics := initMetrics(ctx, otelAgentAddr)
	closeTraces := initTracer(ctx, otelAgentAddr)

	return func() {
		doneCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		// pushes any last exports to the receiver
		closeTraces(doneCtx)
		closeMetrics(doneCtx)
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

// initMetrics initializes a metrics pusher and registers the metrics provider with the global context
func initMetrics(ctx context.Context, otelAgentAddr string) func(context.Context) {
	metricClient := otlpmetricgrpc.NewClient(
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(otelAgentAddr))
	metricExp, err := otlpmetric.New(ctx, metricClient)
	handleErr(err, "Failed to create the collector metric exporter")

	pusher := controller.New(
		processor.NewFactory(
			simple.NewWithHistogramDistribution(),
			metricExp,
		),
		controller.WithExporter(metricExp),
		controller.WithCollectPeriod(2*time.Second),
	)
	global.SetMeterProvider(pusher)

	err = pusher.Start(ctx)
	handleErr(err, "Failed to start metric pusher")

	return func(doneCtx context.Context) {
		// pushes any last exports to the receiver
		if err := pusher.Stop(doneCtx); err != nil {
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

// continuouslySendRequests continuously sends requests to the server and generates random lines of text to be measured
func continuouslySendRequests() {
	var (
		tracer       = otel.Tracer("demo-client-tracer")
		meter        = global.Meter("demo-client-meter")
		instruments  = NewClientInstruments(meter)
		commonLabels = []attribute.KeyValue{
			attribute.String("method", "repl"),
			attribute.String("client", "cli"),
		}
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	)

	for {
		startTime := time.Now()
		ctx, span := tracer.Start(context.Background(), "ExecuteRequest")
		makeRequest(ctx)
		span.End()
		latencyMs := float64(time.Since(startTime)) / 1e6
		nr := int(rng.Int31n(7))
		for i := 0; i < nr; i++ {
			randLineLength := rng.Int63n(999)
			meter.RecordBatch(
				ctx,
				commonLabels,
				instruments.LineCounts.Measurement(1),
				instruments.LineLengths.Measurement(randLineLength),
			)
			fmt.Printf("#%d: LineLength: %dBy\n", i, randLineLength)
		}

		meter.RecordBatch(
			ctx,
			commonLabels,
			instruments.RequestLatency.Measurement(latencyMs),
			instruments.RequestCount.Measurement(1),
		)

		fmt.Printf("Latency: %.3fms\n", latencyMs)
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

// ClientInstruments is a collection of instruments used to measure client requests to the server
type ClientInstruments struct {
	RequestLatency metric.Float64Histogram
	RequestCount   metric.Int64Counter
	LineLengths    metric.Int64Histogram
	LineCounts     metric.Int64Counter
}

// NewClientInstruments takes a meter and builds a set of instruments to be used to measure client requests to the server.
func NewClientInstruments(meter metric.Meter) ClientInstruments {
	return ClientInstruments{
		RequestLatency: metric.Must(meter).
			NewFloat64Histogram(
				"demo_client/request_latency",
				metric.WithDescription("The latency of requests processed"),
			),
		RequestCount: metric.Must(meter).
			NewInt64Counter(
				"demo_client/request_counts",
				metric.WithDescription("The number of requests processed"),
			),
		LineLengths: metric.Must(meter).
			NewInt64Histogram(
				"demo_client/line_lengths",
				metric.WithDescription("The lengths of the various lines in"),
			),
		LineCounts: metric.Must(meter).
			NewInt64Counter(
				"demo_client/line_counts",
				metric.WithDescription("The counts of the lines in"),
			),
	}
}
