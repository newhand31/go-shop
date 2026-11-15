package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func tracedLogger(w io.Writer) *slog.Logger {
	return slog.New(NewLogHandler(slog.NewJSONHandler(w, nil)))
}

func TestLogHandler(t *testing.T) {
	tp, err := NewTracerProvider("none", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()
	wantTrace := span.SpanContext().TraceID().String()

	tests := []struct {
		name      string
		log       func(l *slog.Logger)
		wantTrace bool
	}{
		{"with a span in the context", func(l *slog.Logger) { l.InfoContext(ctx, "x") }, true},
		{"plain Info has no context to read", func(l *slog.Logger) { l.Info("x") }, false},
		{"survives logger.With", func(l *slog.Logger) { l.With("order", 7).InfoContext(ctx, "x") }, true},
		{"survives logger.WithGroup", func(l *slog.Logger) { l.WithGroup("g").InfoContext(ctx, "x") }, true},
		{"background context has no span", func(l *slog.Logger) { l.InfoContext(context.Background(), "x") }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.log(tracedLogger(&buf))
			var line map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &line); err != nil {
				t.Fatalf("not JSON: %v: %s", err, buf.String())
			}
			// With a group the attrs land inside it, so search the raw text for the id instead of one key.
			has := strings.Contains(buf.String(), wantTrace)
			if has != tt.wantTrace {
				t.Errorf("trace id present = %v, want %v: %s", has, tt.wantTrace, buf.String())
			}
		})
	}
}

func TestStdoutExporterWritesTheSpan(t *testing.T) {
	var out bytes.Buffer
	tp, err := NewTracerProvider(ExporterStdout, &out)
	if err != nil {
		t.Fatal(err)
	}
	_, span := tp.Tracer("test").Start(context.Background(), "checkout")
	span.End()
	if err := Shutdown(context.Background(), tp); err != nil { // Shutdown flushes the batch queue
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"Name":"checkout"`) {
		t.Errorf("no span in exporter output: %q", out.String())
	}
}

func TestUnknownExporterIsRejected(t *testing.T) {
	if _, err := NewTracerProvider("jaeger-ish", nil); err == nil {
		t.Fatal("expected an error for an unknown exporter")
	}
}

func TestMetricsObserve(t *testing.T) {
	m := NewMetrics()
	m.Observe("GET", "/products/{id}", 200, 30*time.Millisecond)
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	for _, want := range []string{
		`http_requests_total{method="GET",route="/products/{id}",status="200"} 1`,
		`http_request_duration_seconds_bucket{method="GET",route="/products/{id}",le="0.05"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in:\n%s", want, body)
		}
	}
}
