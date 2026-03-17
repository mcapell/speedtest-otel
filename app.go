package main

import (
	"context"
	"fmt"
	"time"

	"github.com/showwin/speedtest-go/speedtest"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const speedTestTimeout = 5 * time.Minute

type App struct {
	tracer          trace.Tracer
	meter           metric.Meter
	speedtestClient *speedtest.Speedtest
	pingDuration    metric.Float64Histogram
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
		metric.WithUnit("Mbit/s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create upload speed gauge: %w", err)
	}

	downloadSpeed, err := meter.Float64Gauge(
		"speedtest.download.speed",
		metric.WithDescription("Download speed measured during speed test."),
		metric.WithUnit("Mbit/s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create download speed gauge: %w", err)
	}

	return &App{
		tracer:          tracer,
		meter:           meter,
		speedtestClient: speedtest.New(),
		pingDuration:    pingDuration,
		jitter:          jitter,
		uploadSpeed:     uploadSpeed,
		downloadSpeed:   downloadSpeed,
	}, nil
}

func (a *App) runSpeedTest(ctx context.Context) (*speedtest.Server, error) {
	ctx, span := a.tracer.Start(ctx, "runSpeedTest")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, speedTestTimeout)
	defer cancel()

	logger := FromContext(ctx)

	serverList, err := a.speedtestClient.FetchServerListContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching server list: %w", err)
	}

	targets, err := serverList.FindServer(nil)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("server not found")
	}

	target := targets[0]

	logger.Info("start speed test", "server", target.Name)

	if err := target.PingTestContext(ctx, nil); err != nil {
		return nil, fmt.Errorf("error running the ping test: %w", err)
	}
	if err := target.DownloadTestContext(ctx); err != nil {
		return nil, fmt.Errorf("error running download test: %w", err)
	}
	if err := target.UploadTestContext(ctx); err != nil {
		return nil, fmt.Errorf("error running upload test: %w", err)
	}

	logger.Info(fmt.Sprintf("Latency: %s, Jitter: %s, Download: %.2f Mbit/s, Upload: %.2f Mbit/s",
		target.Latency, target.Jitter, target.DLSpeed.Mbps(), target.ULSpeed.Mbps()))

	return target, nil
}

func (a *App) recordMetrics(ctx context.Context, speedTest *speedtest.Server) {
	_, span := a.tracer.Start(ctx, "recordMetrics")
	defer span.End()

	a.pingDuration.Record(ctx, speedTest.Latency.Seconds())
	a.jitter.Record(ctx, speedTest.Jitter.Seconds())
	a.uploadSpeed.Record(ctx, speedTest.ULSpeed.Mbps())
	a.downloadSpeed.Record(ctx, speedTest.DLSpeed.Mbps())
}
