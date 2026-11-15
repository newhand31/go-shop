package handler

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

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"

	"github.com/newhand31/go-shop/internal/platform/telemetry"
	"github.com/newhand31/go-shop/internal/service"
)

// shop builds the same layers main does: otelhttp outside, then the router with RequestLog and Instrument, plus /metrics.
// seenTrace receives the trace id the handler itself sees, so a test can compare it with what reached the log.
func shop(t *testing.T, logs io.Writer) (srv http.Handler, seenTrace *string) {
	return shopWith(t, logs, fakeProducts{items: []service.Product{{ID: 1, Name: "a", PriceTWD: 1}, {ID: 2, Name: "b", PriceTWD: 2}}}, allowAll{})
}

// allowAll and denyAll stand in for the rate limiter.
type allowAll struct{}

func (allowAll) Allow(string) (bool, time.Duration) { return true, 0 }

type denyAll struct{}

func (denyAll) Allow(string) (bool, time.Duration) { return false, time.Second }

// panicProducts blows up inside the handler, to see what the metrics say about a panic.
type panicProducts struct{ Products }

func (panicProducts) Get(context.Context, int) (service.Product, error) { panic("boom") }

// shopWith is main's layering: Instrument first among the extras (before the limiter), the whole thing wrapped by
// WithPaymentCallback, which mounts the API the way main does, with otelhttp outside.
func shopWith(t *testing.T, logs io.Writer, products Products, limiter Limiter) (srv http.Handler, seenTrace *string) {
	t.Helper()
	logger := newTracedLogger(logs)
	tp, err := telemetry.NewTracerProvider("none", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tp.Shutdown(t.Context()) })
	metrics := telemetry.NewMetrics()

	seen := new(string)
	capture := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*seen = trace.SpanContextFromContext(r.Context()).TraceID().String()
			next.ServeHTTP(w, r)
		})
	}
	api := WithPaymentCallback(New(products, fakeCarts{}, fakeOrders{}, logger, capture, Instrument(metrics), RateLimit(limiter, logger)),
		nil, logger, Instrument(metrics))
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", metrics.Handler())
	mux.Handle("/", otelhttp.NewHandler(api, "shop", otelhttp.WithTracerProvider(tp)))
	return mux, seen
}

func get(h http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestRequestLogCarriesTheTraceID(t *testing.T) {
	var logs bytes.Buffer
	h, seen := shop(t, &logs)

	if rec := get(h, "/products/1"); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var line map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(logs.Bytes()), &line); err != nil {
		t.Fatalf("log is not one JSON line: %v\n%s", err, logs.String())
	}
	if *seen == "" || *seen == "00000000000000000000000000000000" {
		t.Fatalf("handler saw no trace id: %q", *seen)
	}
	if line["trace_id"] != *seen {
		t.Errorf("log trace_id = %v, handler saw %q", line["trace_id"], *seen)
	}
	if line["span_id"] == nil || line["span_id"] == "" {
		t.Errorf("log has no span_id: %v", line)
	}
	if line["msg"] != "request" {
		t.Errorf("msg = %v, want the request line", line["msg"])
	}
}

func TestMetricsExposeTheRoutePatternNotThePath(t *testing.T) {
	h, _ := shop(t, io.Discard)
	get(h, "/products/1")
	get(h, "/products/2")
	get(h, "/no/such/page")

	body := get(h, "/metrics").Body.String()
	for _, want := range []string{
		`http_requests_total{method="GET",route="/products/{id}",status="200"} 2`,
		`http_requests_total{method="GET",route="unmatched",status="404"} 1`,
		`http_request_duration_seconds_count{method="GET",route="/products/{id}"} 2`,
		`go_goroutines`, // the runtime collector is on too
	} {
		if !strings.Contains(body, want) {
			t.Errorf("/metrics lacks %q\n%s", want, body)
		}
	}
	for _, bad := range []string{`/products/1"`, `/no/such/page"`} {
		if strings.Contains(body, bad) {
			t.Errorf("/metrics leaks a raw path %q: one series per path would explode", bad)
		}
	}
}

func newTracedLogger(w io.Writer) *slog.Logger {
	return slog.New(telemetry.NewLogHandler(slog.NewJSONHandler(w, nil)))
}

func TestMetricsCountRateLimitedAndPanickedRequests(t *testing.T) {
	denied, _ := shopWith(t, io.Discard, fakeProducts{}, denyAll{})
	if rec := get(denied, "/products/1"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	body := get(denied, "/metrics").Body.String()
	// The limiter answers before chi has matched a route, so a 429 carries the "unmatched" label.
	if want := `http_requests_total{method="GET",route="unmatched",status="429"} 1`; !strings.Contains(body, want) {
		t.Errorf("a 429 is missing from /metrics: want %q\n%s", want, body)
	}

	boom, _ := shopWith(t, io.Discard, panicProducts{}, allowAll{})
	if rec := get(boom, "/products/1"); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 from Recover", rec.Code)
	}
	body = get(boom, "/metrics").Body.String()
	if want := `http_requests_total{method="GET",route="/products/{id}",status="500"} 1`; !strings.Contains(body, want) {
		t.Errorf("a panic is missing from /metrics: want %q\n%s", want, body)
	}
}
