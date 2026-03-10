package main

import (
	"context"
	"log/slog"
	"testing"
)

func TestFromContext_returnsDefaultLoggerWhenNotSet(t *testing.T) {
	logger := FromContext(context.Background())
	if logger == nil {
		t.Fatal("expected a non-nil logger, got nil")
	}
}

func TestWithContext_roundtrip(t *testing.T) {
	expected := slog.Default()
	ctx := WithContext(context.Background(), expected)

	got := FromContext(ctx)
	if got != expected {
		t.Fatalf("expected logger %v, got %v", expected, got)
	}
}
