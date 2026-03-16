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
	tests := []struct {
		name    string
		server  *speedtest.Server
		metrics []struct {
			name     string
			expected float64
			isGauge  bool
		}
	}{
		{
			name: "all metrics recorded",
			server: &speedtest.Server{
				Latency: 20 * time.Millisecond,
				Jitter:  5 * time.Millisecond,
				DLSpeed: speedtest.ByteRate(100.0),
				ULSpeed: speedtest.ByteRate(50.0),
			},
			metrics: []struct {
				name     string
				expected float64
				isGauge  bool
			}{
				{name: "latency", expected: 20},
				{name: "jitter", expected: 5},
				{name: "download_speed", expected: speedtest.ByteRate(100.0).Mbps(), isGauge: true},
				{name: "upload_speed", expected: speedtest.ByteRate(50.0).Mbps(), isGauge: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := sdkmetric.NewManualReader()
			mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			app := &App{
				tracer: noop.NewTracerProvider().Tracer(""),
				meter:  mp.Meter("test"),
			}

			if err := app.recordMetrics(context.Background(), tt.server); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var rm metricdata.ResourceMetrics
			if err := reader.Collect(context.Background(), &rm); err != nil {
				t.Fatalf("failed to collect metrics: %v", err)
			}

			for _, mc := range tt.metrics {
				t.Run(mc.name, func(t *testing.T) {
					m := findMetric(t, rm, mc.name)
					if mc.isGauge {
						data := m.Data.(metricdata.Gauge[float64])
						if len(data.DataPoints) != 1 {
							t.Fatalf("expected 1 data point, got %d", len(data.DataPoints))
						}
						if data.DataPoints[0].Value != mc.expected {
							t.Errorf("expected %v, got %v", mc.expected, data.DataPoints[0].Value)
						}
					} else {
						data := m.Data.(metricdata.Histogram[float64])
						if len(data.DataPoints) != 1 {
							t.Fatalf("expected 1 data point, got %d", len(data.DataPoints))
						}
						if data.DataPoints[0].Sum != mc.expected {
							t.Errorf("expected sum %v, got %v", mc.expected, data.DataPoints[0].Sum)
						}
					}
				})
			}
		})
	}
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
