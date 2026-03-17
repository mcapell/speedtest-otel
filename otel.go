package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	shutdownTimeout      = 10 * time.Second
	metricExportInterval = 5 * time.Second
)

func initTelemetry(ctx context.Context, logger *slog.Logger, serviceName string) (*App, func(), error) {
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize otel resource: %w", err)
	}

	traceExp, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
	)

	metricExp, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, nil, fmt.Errorf("failed to initialize metric exporter: %w", err)
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp,
			sdkmetric.WithInterval(metricExportInterval),
		)),
		sdkmetric.WithResource(res),
	)

	app, err := newApp(tp.Tracer(serviceName), mp.Meter(serviceName))
	if err != nil {
		_ = tp.Shutdown(ctx)
		_ = mp.Shutdown(ctx)
		return nil, nil, fmt.Errorf("failed to initialize instruments: %w", err)
	}

	shutdown := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := tp.Shutdown(shutdownCtx); err != nil {
			logger.Error("failed to shut down trace provider", "error", err)
		}
		if err := mp.Shutdown(shutdownCtx); err != nil {
			logger.Error("failed to shut down meter provider", "error", err)
		}
	}

	return app, shutdown, nil
}
