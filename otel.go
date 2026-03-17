package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	shutdownTimeout      = 10 * time.Second
	metricExportInterval = 5 * time.Second
	defaultOTLPEndpoint  = "localhost:4317"
)

// newSharedGRPCConn creates a single gRPC client connection that is shared by
// both the trace and metric exporters, avoiding duplicate TCP connections to
// the same collector endpoint. TLS is used by default; set
// OTEL_EXPORTER_OTLP_METRICS_INSECURE=true to disable it.
func newSharedGRPCConn() (*grpc.ClientConn, error) {
	creds := credentials.NewTLS(nil)
	if os.Getenv("OTEL_EXPORTER_OTLP_METRICS_INSECURE") == "true" {
		creds = insecure.NewCredentials()
	}

	ep := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if ep == "" {
		ep = defaultOTLPEndpoint
	}
	ep = strings.TrimPrefix(ep, "https://")
	ep = strings.TrimPrefix(ep, "http://")

	conn, err := grpc.NewClient(ep, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}
	return conn, nil
}

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

	conn, err := newSharedGRPCConn()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize gRPC connection: %w", err)
	}

	traceExp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("failed to initialize trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
	)

	metricExp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		_ = tp.Shutdown(ctx)
		_ = conn.Close()
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
		_ = conn.Close()
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
		if err := conn.Close(); err != nil {
			logger.Error("failed to close gRPC connection", "error", err)
		}
	}

	return app, shutdown, nil
}
