package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// RequestObserver is what Instrument needs, defined here by the consumer. telemetry.Metrics fits without saying so.
type RequestObserver interface {
	Observe(method, route string, status int, d time.Duration)
}

// unmatchedRoute is the label for every request no route claimed (404s, scanners probing /wp-admin).
const unmatchedRoute = "unmatched"

// Instrument tells obs how each request ended: method, route pattern, status and duration.
//
// The label is the route PATTERN (/products/{id}), read from chi after the request ran. Chi only knows which route
// matched once routing is done, so the read happens after next, not before. The raw path would be a metrics
// cardinality bomb: one new time series per id, and a scanner can invent paths faster than we can store them.
func Instrument(obs RequestObserver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			route := unmatchedRoute
			if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
				route = rc.RoutePattern()
			}
			obs.Observe(r.Method, route, rec.status, time.Since(start))
		})
	}
}
