package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger := initLogger()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx = WithContext(ctx, logger)

	app, shutdown, err := initTelemetry(ctx, logger, "speedtest")
	if err != nil {
		logger.Error("open-telemetry setup", "error", err)
		os.Exit(1)
	}

	err = run(ctx, app)
	shutdown()

	if err != nil {
		logger.Error("speedtest failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, app *App) error {
	ctx, span := app.tracer.Start(ctx, "speedtest")
	defer span.End()

	speedTest, err := app.runSpeedTest(ctx)
	if err != nil {
		return fmt.Errorf("speed test: %w", err)
	}

	app.recordMetrics(ctx, speedTest)
	return nil
}
