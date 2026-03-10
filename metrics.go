package main

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/metric"
	"github.com/showwin/speedtest-go/speedtest"
)

func recordMetrics(ctx context.Context, speedTest *speedtest.Server) error {
	_, span := tracer.Start(ctx, "recordMetrics")
	defer span.End()

	latency, err := meter.Float64Histogram(
		"latency",
		metric.WithDescription("Speed test latency in microseconds."),
		metric.WithUnit("us"),
	)
	if err != nil {
		return fmt.Errorf("failed to create latency histogram: %w", err)
	}

	uploadSpeed, err := meter.Float64Gauge(
		"upload_speed",
		metric.WithDescription("Upload speed in bytes/second."),
		metric.WithUnit("By/s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create upload speed gauge: %w", err)
	}

	downloadSpeed, err := meter.Float64Gauge(
		"download_speed",
		metric.WithDescription("Download speed in bytes/second."),
		metric.WithUnit("By/s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create download speed gauge: %w", err)
	}

	latency.Record(ctx, float64(speedTest.Latency.Microseconds()))
	uploadSpeed.Record(ctx, float64(speedTest.ULSpeed))
	downloadSpeed.Record(ctx, float64(speedTest.DLSpeed))

	return nil
}
