package main

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type App struct {
	tracer          trace.Tracer
	meter           metric.Meter
	pingDuration metric.Float64Histogram
	jitter          metric.Float64Gauge
	uploadSpeed     metric.Float64Gauge
	downloadSpeed   metric.Float64Gauge
}

func newApp(tracer trace.Tracer, meter metric.Meter) (*App, error) {
	pingDuration, err := meter.Float64Histogram(
		"speedtest.ping.duration",
		metric.WithDescription("Latency measured during ping test."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ping duration histogram: %w", err)
	}

	jitter, err := meter.Float64Gauge(
		"speedtest.ping.jitter",
		metric.WithDescription("Jitter measured during ping test."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ping jitter gauge: %w", err)
	}

	uploadSpeed, err := meter.Float64Gauge(
		"speedtest.upload.speed",
		metric.WithDescription("Upload speed measured during speed test."),
		metric.WithUnit("By/s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create upload speed gauge: %w", err)
	}

	downloadSpeed, err := meter.Float64Gauge(
		"speedtest.download.speed",
		metric.WithDescription("Download speed measured during speed test."),
		metric.WithUnit("By/s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create download speed gauge: %w", err)
	}

	return &App{
		tracer:        tracer,
		meter:         meter,
		pingDuration:  pingDuration,
		jitter:        jitter,
		uploadSpeed:   uploadSpeed,
		downloadSpeed: downloadSpeed,
	}, nil
}

func newResource(serviceName string) (*resource.Resource, error) {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("error merging provider resource: %w", err)
	}
	return r, nil
}

func newTraceProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize trace exporter: %w", err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	), nil
}

func newMeterProvider(ctx context.Context, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	exp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metric exporter: %w", err)
	}

	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)),
		sdkmetric.WithResource(res),
	), nil
}

func initTelemetry(ctx context.Context, serviceName string) (*App, func(), error) {
	res, err := newResource(serviceName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize otel resource: %w", err)
	}

	tp, err := newTraceProvider(ctx, res)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize otel trace provider: %w", err)
	}

	mp, err := newMeterProvider(ctx, res)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize otel meter provider: %w", err)
	}

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)

	app, err := newApp(tp.Tracer(serviceName), mp.Meter(serviceName))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize instruments: %w", err)
	}

	return app, func() {
		_ = tp.Shutdown(ctx)
		_ = mp.Shutdown(ctx)
	}, nil
}
