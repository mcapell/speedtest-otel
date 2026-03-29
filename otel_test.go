package main

import (
	"testing"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func TestNewResource(t *testing.T) {
	res, err := newResource("test-service")
	if err != nil {
		t.Fatalf("newResource failed: %v", err)
	}

	val, ok := res.Set().Value(semconv.ServiceNameKey)
	if !ok {
		t.Fatal("service.name attribute not found")
	}
	if val.AsString() != "test-service" {
		t.Errorf("expected service name %q, got %q", "test-service", val.AsString())
	}
}

func TestNewResource_schemaURLMatchesDefault(t *testing.T) {
	res, err := newResource("test-service")
	if err != nil {
		t.Fatalf("newResource failed: %v", err)
	}

	if res.SchemaURL() != resource.Default().SchemaURL() {
		t.Errorf("schema URL %q does not match default %q", res.SchemaURL(), resource.Default().SchemaURL())
	}
}
