package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestIncomingTraceparentIsContinued(t *testing.T) {
	installPropagator()
	const traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	tests := []struct {
		name        string
		traceparent string
		wantSame    bool
	}{
		{"valid header keeps the caller's trace id", "00-" + traceID + "-00f067aa0ba902b7-01", true},
		{"no header starts a new trace", "", false},
		{"garbage header starts a new trace", "not-a-traceparent", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got trace.TraceID
			inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = trace.SpanContextFromContext(r.Context()).TraceID()
			})
			h := otelhttp.NewHandler(inner, "test", otelhttp.WithTracerProvider(sdktrace.NewTracerProvider()))
			req := httptest.NewRequest(http.MethodGet, "/products", nil)
			if tt.traceparent != "" {
				req.Header.Set("traceparent", tt.traceparent)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			if !got.IsValid() {
				t.Fatal("request has no trace id at all")
			}
			if same := got.String() == traceID; same != tt.wantSame {
				t.Errorf("trace id = %s, same as header = %v, want %v", got, same, tt.wantSame)
			}
		})
	}
}

func TestMailerFollowsSettings(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	t.Setenv("SMTP_HOST", "")
	if _, ok := mailer(logger).(logMailer); !ok {
		t.Error("with no SMTP_HOST the mailer must be the logger")
	}
	t.Setenv("SMTP_HOST", "mail.test")
	if _, ok := mailer(logger).(logMailer); ok {
		t.Error("with SMTP_HOST set the mailer must send by SMTP")
	}
}
