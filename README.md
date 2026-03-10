# Speedtest OpenTelemetry Exporter

This application performs an internet speed test using the `speedtest-go` library and exports the results as OpenTelemetry metrics and traces.

## Features

- Measures latency, download, and upload speeds.
- Exports metrics and traces via OTLP to any OpenTelemetry-compatible backend (Prometheus, Grafana, Datadog, etc.)

## Prerequisites

- Go 1.23 or later
- An OpenTelemetry collector or OTLP-compatible backend

## Configuration

The application uses the standard OpenTelemetry environment variables to configure the exporter:

| Variable | Description | Default |
|---|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP collector endpoint | `localhost:4317` |
| `OTEL_EXPORTER_OTLP_INSECURE` | Disable TLS (set to `true` for local collectors) | `false` |

Example:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://your-otel-collector:4317
```

## Usage

Run the application:

```bash
go run .
```

Or the docker image:

https://hub.docker.com/r/mcapell/speedtest
