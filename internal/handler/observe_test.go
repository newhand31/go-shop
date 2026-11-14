package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"

	"github.com/newhand31/go-shop/internal/platform/telemetry"
	"github.com/newhand31/go-shop/internal/service"
)

// shop builds the same layers main does: otelhttp outside, then the router with RequestLog and Instrument, plus /metrics.
// seenTrace receives the trace id the handler itself sees, so a test can compare it with what reached the log.
func shop(t *testing.T, logs io.Writer) (srv http.Handler, seenTrace *string) {
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
	api := New(fakeProducts{items: []service.Product{{ID: 1, Name: "a", PriceTWD: 1}, {ID: 2, Name: "b", PriceTWD: 2}}},
		fakeCarts{}, fakeOrders{}, logger, capture, Instrument(metrics))
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
