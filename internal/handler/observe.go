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
			finished := false
			// A panic skips every line after next, so the deferred call is the only way to count it. It counts a 500 (the
			// status Recover, one layer out, is about to write) and does not recover: the panic keeps travelling.
			defer func() {
				if !finished {
					rec.status = http.StatusInternalServerError
				}
				obs.Observe(r.Method, routeLabel(r), rec.status, time.Since(start))
			}()
			next.ServeHTTP(rec, r)
			finished = true
		})
	}
}

// routeLabel is the pattern chi matched. The "/*" that Mount("/", api) registers is a catch-all, not a real route:
// it is what every unknown path matches, so it becomes "unmatched" like an empty pattern.
func routeLabel(r *http.Request) string {
	if rc := chi.RouteContext(r.Context()); rc != nil {
		if p := rc.RoutePattern(); p != "" && p != "/*" {
			return p
		}
	}
	return unmatchedRoute
}
