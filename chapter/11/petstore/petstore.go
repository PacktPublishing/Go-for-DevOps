package main

import (
	"context"
	"flag"
	stdlog "log"
	"os"
	"strconv"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/11/petstore/internal/server"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/11/petstore/internal/server/log"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/11/petstore/internal/server/storage/mem"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/11/petstore/internal/server/telemetry/metrics"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/11/petstore/internal/server/telemetry/tracing"

	//grpcotel "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc"
	"go.opentelemetry.io/otel/sdk/trace"
)

// General service flags.
var (
	addr = flag.String("addr", "0.0.0.0:6742", "The address to run the service on.")
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

// otelExporter determines if the flags are set to export tracing information
// to a destination. If so, we return the arguments needed for that exporter.
func otelExporter() tracing.Exporter {
	if tooManyTrue(*localDebug, *fileDebug, *grpcTraces) {
		log.Logger.Fatalf("cannot set more than one from this list: localDebug, fileDebug, grpcTraces")
	}

	switch {
	case *localDebug:
		return tracing.Stderr{}
	case *fileDebug != "":
		return tracing.File{Path: *fileDebug}
	case *grpcTraces:
		return tracing.OTELGRPC{Addr: *otelAddr}
	}
	return tracing.Stderr{}
}

// otelController is similar to otelExporter except it sets up arguments for metric
// exporting.
func otelController() metrics.Controller {
	if *otelAddr != "" {
		return metrics.OTELGRPC{Addr: *otelAddr}
	}
	return nil
}

// setSampling checks flags and then sets our tracing sampling rate.
func setSampling() {
	switch *traceSampling {
	case "never":
		tracing.Sampler.Switch(trace.NeverSample())
		return
	case "always":
		tracing.Sampler.Switch(trace.AlwaysSample())
		return
	default:
		if f, err := strconv.ParseFloat(*traceSampling, 64); err == nil {
			tracing.Sampler.Switch(trace.TraceIDRatioBased(f))
			return
		}
	}
	log.Logger.Fatalf("traceSampling=%s is not a valid value", *traceSampling)
}

// tooManyTrue is given a list of bool or string types. A string type that
// is non-empty string is considered true. If more than one value is true,
// this returns true. Otherwise it returns false.
func tooManyTrue(truths ...interface{}) bool {
	set := false
	for _, t := range truths {
		switch v := t.(type) {
		case bool:
			if v && set {
				return true
			}
			if v {
				set = true
			}
		case string:
			if v != "" && set {
				return true
			}
			if v != "" {
				set = true
			}
		default:
			panic("not a bool or string")
		}
	}
	return false
}

func main() {
	flag.Parse()
	log.Logger.Println("Flags values")
	log.Logger.Println("-----------------------------------")
	flag.VisitAll(func(f *flag.Flag) {
		log.Logger.Printf("%s: %s\n", f.Name, f.Value)
	})
	log.Logger.Println("-----------------------------------")

	stdlog.SetFlags(stdlog.LstdFlags | stdlog.Lshortfile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Logger.SetFlags(stdlog.LstdFlags | stdlog.Lshortfile)

	ctx := context.Background()

	// Setup for OTEL tracing.
	setSampling()
	e := otelExporter()
	if e != nil {
		stop, err := tracing.Start(ctx, e)
		if err != nil {
			log.Logger.Fatalf("problem starting telemetry: %s", err)
		}
		defer stop()
	}

	// Setup for OTEL metrics.
	c := otelController()
	if c != nil {
		stop, err := metrics.Start(ctx, c)
		if err != nil {
			log.Logger.Fatal(err)
		}
		defer stop()
	}

	// Setup for the service.
	store := mem.New()

	s, err := server.New(
		*addr,
		store,
		server.WithGRPCOpts(
		//grpc.UnaryInterceptor(grpcotel.UnaryServerInterceptor(tracing.Tracer)),
		//grpc.StreamInterceptor(grpcotel.StreamServerInterceptor(tracing.Tracer)),
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

	log.Logger.Println("Server exited with error: ", <-done)
}
