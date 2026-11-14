// Package telemetry wires the three signals of episode 11: traces (OpenTelemetry), metrics (Prometheus) and logs that
// carry the trace id (slog). Nothing here needs an external collector: spans go to stderr or nowhere, metrics are
// pulled from /metrics. Pointing the exporter at a real collector later changes this file, not the callers.
package telemetry

import (
	"context"
	"fmt"
	"io"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Exporter names for OTEL_TRACES_EXPORTER (the variable name the OpenTelemetry spec uses).
const (
	ExporterNone   = "none"   // default: spans exist (so trace ids exist) but nobody receives them
	ExporterStdout = "stdout" // one JSON document per span, handy for a demo
)

// NewTracerProvider builds the provider for the named exporter. An empty name means ExporterNone.
//
// The sampler is AlwaysSample: every request gets a real trace id, which is what the log lines need. A production
// service with real traffic would sample a fraction and keep the id on every log line anyway.
func NewTracerProvider(exporter string, out io.Writer) (*sdktrace.TracerProvider, error) {
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource.NewSchemaless(attribute.String("service.name", "go-shop"))),
	}
	switch exporter {
	case "", ExporterNone:
	case ExporterStdout:
		exp, err := stdouttrace.New(stdouttrace.WithWriter(out))
		if err != nil {
			return nil, fmt.Errorf("stdout trace exporter: %w", err)
		}
		opts = append(opts, sdktrace.WithBatcher(exp))
	default:
		return nil, fmt.Errorf("unknown OTEL_TRACES_EXPORTER %q (use %q or %q)", exporter, ExporterNone, ExporterStdout)
	}
	return sdktrace.NewTracerProvider(opts...), nil
}

// Shutdown flushes the spans still sitting in the batch queue. Call it last, after the server stopped taking requests.
func Shutdown(ctx context.Context, tp *sdktrace.TracerProvider) error {
	return tp.Shutdown(ctx)
}
