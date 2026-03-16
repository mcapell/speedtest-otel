package main

import (
	"context"
	"os"
)

func main() {
	logger := initLogger()
	ctx := WithContext(context.Background(), logger)

	app, shutdown, err := initTelemetry(ctx, "speedtest")
	if err != nil {
		logger.Error("open-telemetry setup", "error", err)
		os.Exit(1)
	}
	defer shutdown()

	ctx, span := app.tracer.Start(ctx, "speedtest")
	defer span.End()

	speedTest, err := app.runSpeedTest(ctx)
	if err != nil {
		logger.Error("speed test failed", "error", err)
		os.Exit(1)
	}

	if err := app.recordMetrics(ctx, speedTest); err != nil {
		logger.Error("metrics recording failed", "error", err)
		os.Exit(1)
	}
}
