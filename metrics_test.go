package main

import (
	"context"
	"testing"
	"time"

	"github.com/showwin/speedtest-go/speedtest"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestRecordMetrics(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	app := &App{
		tracer: noop.NewTracerProvider().Tracer(""),
		meter:  mp.Meter("test"),
	}

	server := &speedtest.Server{}
	server.Latency = 20 * time.Millisecond
	server.DLSpeed = speedtest.ByteRate(100.0)
	server.ULSpeed = speedtest.ByteRate(50.0)

	if err := app.recordMetrics(context.Background(), server); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	t.Run("latency", func(t *testing.T) {
		m := findMetric(t, rm, "latency")
		data := m.Data.(metricdata.Histogram[float64])
		if len(data.DataPoints) != 1 {
			t.Fatalf("expected 1 data point, got %d", len(data.DataPoints))
		}
		expected := float64(server.Latency.Milliseconds())
		if data.DataPoints[0].Sum != expected {
			t.Errorf("expected sum %v, got %v", expected, data.DataPoints[0].Sum)
		}
	})

	t.Run("download_speed", func(t *testing.T) {
		m := findMetric(t, rm, "download_speed")
		data := m.Data.(metricdata.Gauge[float64])
		if len(data.DataPoints) != 1 {
			t.Fatalf("expected 1 data point, got %d", len(data.DataPoints))
		}
		if data.DataPoints[0].Value != server.DLSpeed.Mbps() {
			t.Errorf("expected %v, got %v", server.DLSpeed.Mbps(), data.DataPoints[0].Value)
		}
	})

	t.Run("upload_speed", func(t *testing.T) {
		m := findMetric(t, rm, "upload_speed")
		data := m.Data.(metricdata.Gauge[float64])
		if len(data.DataPoints) != 1 {
			t.Fatalf("expected 1 data point, got %d", len(data.DataPoints))
		}
		if data.DataPoints[0].Value != server.ULSpeed.Mbps() {
			t.Errorf("expected %v, got %v", server.ULSpeed.Mbps(), data.DataPoints[0].Value)
		}
	})
}

func findMetric(t *testing.T, rm metricdata.ResourceMetrics, name string) metricdata.Metrics {
	t.Helper()
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				return m
			}
		}
	}
	t.Fatalf("metric %q not found", name)
	return metricdata.Metrics{}
}
