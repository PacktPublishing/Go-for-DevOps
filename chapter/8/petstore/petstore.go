package main

import (
	"context"
	"flag"
	"os"
	"strconv"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/petstore/internal/server"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/petstore/internal/server/log"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/petstore/internal/server/storage/mem"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/petstore/internal/server/telemetry"

	//grpcotel "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc"
	"go.opentelemetry.io/otel/sdk/trace"
)

// General service flags.
var (
	addr = flag.String("addr", "127.0.0.1:6742", "The address to run the service on.")
)

// Flags are related to OTEL tracing.
var (
	localDebug = flag.Bool("localDebug", false, "If true, OTEL traces are sent to the console")
	fileDebug  = flag.String("fileDebug", "", "If set, OTEL traces are written to the file path provided")
	grpcTraces = flag.Bool("grpcTraces", false, "Our traces are exported via gRPC. Must set otelAddr.")

	traceSampling = flag.String("traceSampling", "never", "Sets the sampling type. By default we never sample unless it is requested by the client."+
		"Valid values are: 'never', 'always' and '[float]', where float is a floating point value where any value over 1 is all.",
	)
)

// These flags relate to exporting our Open Telemetry traces via gRPC.
var (
	otelAddr = flag.String("otelAddr", "", "The address for our OpenTelemetry agent. If not set, looks for Env variable 'OTEL_EXPORTER_OTLP_ENDPOINT'. If not set defaults to 0.0.0:4317")
)

func init() {
	if *otelAddr == "" {
		addr, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
		if !ok {
			addr = "0.0.0.0:4317"
		}
		*otelAddr = addr
	}
}

func otelExporter() telemetry.Exporter {
	if tooManyTrue(*localDebug, *fileDebug, *grpcTraces) {
		log.Logger.Fatalf("cannot set more than one from this list: localDebug, fileDebug, grpcTraces")
	}

	switch {
	case *localDebug:
		return telemetry.Stderr{}
	case *fileDebug != "":
		return telemetry.File{Path: *fileDebug}
	case *grpcTraces:
		return telemetry.OTELGRPC{Addr: *otelAddr}
	}
	return nil
}

func setSampling() {
	switch *traceSampling {
	case "never":
		telemetry.Sampler.Switch(trace.NeverSample())
		return
	case "always":
		telemetry.Sampler.Switch(trace.AlwaysSample())
		return
	default:
		if f, err := strconv.ParseFloat(*traceSampling, 64); err == nil {
			telemetry.Sampler.Switch(trace.TraceIDRatioBased(f))
			return
		}
	}
	log.Logger.Fatalf("traceSampling=%s is not a valid value", *traceSampling)
}

func tooManyTrue(truths ...interface{}) bool {
	set := false
	for _, t := range truths {
		switch v := t.(type) {
		case bool:
			if v && set {
				return true
			}
			set = true
		case string:
			if v != "" && set {
				return true
			}
			set = true
		}
	}
	return false
}

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	ctx := context.Background()

	setSampling()
	e := otelExporter()
	if e != nil {
		stop, err := telemetry.Start(ctx, e)
		if err != nil {
			log.Logger.Fatalf("problem starting telemetry: %s", err)
		}
		defer stop()
	}

	store := mem.New()

	s, err := server.New(
		*addr,
		store,
		server.WithGRPCOpts(
		//grpc.UnaryInterceptor(grpcotel.UnaryServerInterceptor(telemetry.Tracer)),
		//grpc.StreamInterceptor(grpcotel.StreamServerInterceptor(telemetry.Tracer)),
		),
	)
	if err != nil {
		panic(err)
	}

	done := make(chan error, 1)

	log.Logger.Println("Starting server at: ", *addr)
	go func() {
		defer close(done)
		done <- s.Start()
	}()

	err = <-done
	log.Logger.Println("Server exited with error: ", err)
}
