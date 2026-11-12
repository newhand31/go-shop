package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/newhand31/go-shop/internal/service"
)

type fakeLister struct {
	items []service.Product
	err   error
}

func (f fakeLister) List(context.Context) ([]service.Product, error) { return f.items, f.err }

func TestRoutes(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	two := []service.Product{{ID: 1, Name: "a", PriceTWD: 1}, {ID: 2, Name: "b", PriceTWD: 2}}
	tests := []struct {
		name       string
		method     string
		path       string
		lister     fakeLister
		wantStatus int
		wantCount  int // -1 means do not check the JSON
	}{
		{"healthz", http.MethodGet, "/healthz", fakeLister{}, http.StatusOK, -1},
		{"products with data", http.MethodGet, "/products", fakeLister{items: two}, http.StatusOK, 2},
		{"products empty list returns []", http.MethodGet, "/products", fakeLister{}, http.StatusOK, 0},
		{"products service error", http.MethodGet, "/products", fakeLister{err: errors.New("x")}, http.StatusInternalServerError, -1},
		{"unknown path", http.MethodGet, "/nope", fakeLister{}, http.StatusNotFound, -1},
		{"products rejects POST", http.MethodPost, "/products", fakeLister{}, http.StatusMethodNotAllowed, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			New(tt.lister, logger).ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantCount < 0 {
				return
			}
			var got []map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("body is not a JSON array: %v (%s)", err, rec.Body.String())
			}
			if len(got) != tt.wantCount {
				t.Errorf("count = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}
