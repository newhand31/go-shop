// Package handler is the HTTP layer: parse the request, call the service, write the response. No SQL here.
package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/newhand31/go-shop/internal/service"
)

// ProductLister is the service capability the handler needs, defined by the consumer.
type ProductLister interface {
	List(ctx context.Context) ([]service.Product, error)
}

type productJSON struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	PriceTWD int    `json:"price_twd"`
}

// New builds the router. A middleware is just func(http.Handler) http.Handler.
func New(products ProductLister, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(RequestLog(logger))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Get("/products", func(w http.ResponseWriter, req *http.Request) {
		items, err := products.List(req.Context())
		if err != nil {
			logger.Error("list products failed", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		out := make([]productJSON, 0, len(items))
		for _, p := range items {
			out = append(out, productJSON{ID: p.ID, Name: p.Name, PriceTWD: p.PriceTWD})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(out); err != nil {
			logger.Error("write response failed", "err", err)
		}
	})
	return r
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// RequestLog logs each request's method, path, status code and duration with slog.
func RequestLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}
