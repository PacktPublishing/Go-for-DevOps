package metrics

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
)

// Controller represents the controller to send metrics to.
type Controller interface {
	isController()
}

// OTELGRPC represents exporting to the "go.opentelemetry.io/otel/sdk/metric/controller/basic" controller.
type OTELGRPC struct {
	// Addr is the local address to export on.
	Addr string
}

func (o OTELGRPC) isController() {}

// Stop is used to stop OTEL metric handling.
type Stop func()

// Start is used to start OTEL metric handling.
func Start(ctx context.Context, c Controller) (Stop, error) {
	control, err := newController(ctx, c)
	if err != nil {
		return nil, err
	}

	err = control.Start(ctx)
	if err != nil {
		return nil, err
	}
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()
		if err := control.Stop(ctx); err != nil {
			otel.Handle(err)
		}
	}, nil
}

func newController(ctx context.Context, c Controller) (*basic.Controller, error) {
	switch v := c.(type) {
	case OTELGRPC:
		return otelGRPC(ctx, v)
	}
	return nil, fmt.Errorf("%T is not a valid Controller", c)
}

func otelGRPC(ctx context.Context, args OTELGRPC) (*basic.Controller, error) {
	metricClient := otlpmetricgrpc.NewClient(
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(args.Addr),
	)
	metricExp, err := otlpmetric.New(ctx, metricClient)
	if err != nil {
		return nil, fmt.Errorf("Failed to create the collector metric exporter")
	}

	pusher := basic.New(
		processor.NewFactory(
			simple.NewWithHistogramDistribution(),
			metricExp,
		),
		basic.WithExporter(metricExp),
		basic.WithCollectPeriod(10*time.Second),
	)
	global.SetMeterProvider(pusher)
	return pusher, nil
}
