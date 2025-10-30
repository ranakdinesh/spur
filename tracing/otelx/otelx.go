package otelx

import (
	"context"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type Options struct {
	ServiceName  string  // required
	Environment  string  // e.g., "development" | "production"
	OTLPEndpoint string  // e.g., http://otel-collector:4318
	SampleRatio  float64 // 0..1 (default 1 in dev, 0.1 in prod if zero)
}

func Start(ctx context.Context, opt Options) (func(context.Context) error, error) {
	if opt.ServiceName == "" {
		opt.ServiceName = os.Getenv("OTEL_SERVICE_NAME")
	}
	if opt.Environment == "" {
		opt.Environment = os.Getenv("APP_ENV")
	}
	if opt.OTLPEndpoint == "" {
		opt.OTLPEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}

	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(opt.OTLPEndpoint), // accepts http(s)://host:4318
	)
	if err != nil {
		return nil, err
	}

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleDefault(opt)))
	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(opt.ServiceName),
			semconv.DeploymentEnvironmentKey.String(opt.Environment),
		),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exp,
			sdktrace.WithBatchTimeout(2*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
	)
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}

func sampleDefault(opt Options) float64 {
	if opt.SampleRatio > 0 {
		return opt.SampleRatio
	}
	if opt.Environment == "production" {
		return 0.1
	}
	return 1.0
}
