package main

import (
	"context"
	"fmt"

	"github.com/showwin/speedtest-go/speedtest"
	"go.opentelemetry.io/otel/metric"
)

func (a *App) recordMetrics(ctx context.Context, speedTest *speedtest.Server) error {
	_, span := a.tracer.Start(ctx, "recordMetrics")
	defer span.End()

	latency, err := a.meter.Float64Histogram(
		"latency",
		metric.WithDescription("Speed test latency in milliseconds."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return fmt.Errorf("failed to create latency histogram: %w", err)
	}

	uploadSpeed, err := a.meter.Float64Gauge(
		"upload_speed",
		metric.WithDescription("Upload speed in megabits/second."),
		metric.WithUnit("Mbit/s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create upload speed gauge: %w", err)
	}

	downloadSpeed, err := a.meter.Float64Gauge(
		"download_speed",
		metric.WithDescription("Download speed in megabits/second."),
		metric.WithUnit("Mbit/s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create download speed gauge: %w", err)
	}

	jitter, err := a.meter.Float64Histogram(
		"jitter",
		metric.WithDescription("Speed test jitter in milliseconds."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return fmt.Errorf("failed to create jitter histogram: %w", err)
	}

	latency.Record(ctx, float64(speedTest.Latency.Milliseconds()))
	uploadSpeed.Record(ctx, speedTest.ULSpeed.Mbps())
	downloadSpeed.Record(ctx, speedTest.DLSpeed.Mbps())
	jitter.Record(ctx, float64(speedTest.Jitter.Milliseconds()))

	return nil
}
