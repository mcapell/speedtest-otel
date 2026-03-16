package main

import (
	"context"

	"github.com/showwin/speedtest-go/speedtest"
)

func (a *App) recordMetrics(ctx context.Context, speedTest *speedtest.Server) error {
	_, span := a.tracer.Start(ctx, "recordMetrics")
	defer span.End()

	a.pingDuration.Record(ctx, speedTest.Latency.Seconds())
	a.jitter.Record(ctx, speedTest.Jitter.Seconds())
	a.uploadSpeed.Record(ctx, float64(speedTest.ULSpeed))
	a.downloadSpeed.Record(ctx, float64(speedTest.DLSpeed))

	return nil
}
