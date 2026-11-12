// Package handler 是 HTTP 層：解析請求、呼叫 service、寫回應。不碰 SQL。
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

// ProductLister 是 handler 需要的 service 能力，由使用端定義。
type ProductLister interface {
	List(ctx context.Context) ([]service.Product, error)
}

type productJSON struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	PriceTWD int    `json:"price_twd"`
}

// New 組出 router。middleware 就是 func(http.Handler) http.Handler。
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

// RequestLog 以 slog 記錄每個請求的方法、路徑、狀態碼與耗時。
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
