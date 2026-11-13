package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/newhand31/go-shop/internal/api"
	"github.com/newhand31/go-shop/internal/service"
)

type fakeProducts struct {
	items []service.Product
	err   error
}

func (f fakeProducts) List(context.Context) ([]service.Product, error) { return f.items, f.err }

func (f fakeProducts) Get(_ context.Context, id int) (service.Product, error) {
	if f.err != nil {
		return service.Product{}, f.err
	}
	for _, p := range f.items {
		if p.ID == id {
			return p, nil
		}
	}
	return service.Product{}, fmt.Errorf("get product %d: %w", id, service.ErrProductNotFound)
}

func quietLogger() *slog.Logger { return slog.New(slog.NewJSONHandler(io.Discard, nil)) }

func TestRoutes(t *testing.T) {
	two := []service.Product{{ID: 1, Name: "a", PriceTWD: 1}, {ID: 2, Name: "b", PriceTWD: 2}}
	tests := []struct {
		name       string
		method     string
		path       string
		products   fakeProducts
		wantStatus int
		wantCount  int // -1 means do not check the JSON array
	}{
		{"healthz", http.MethodGet, "/healthz", fakeProducts{}, http.StatusOK, -1},
		{"products with data", http.MethodGet, "/products", fakeProducts{items: two}, http.StatusOK, 2},
		{"products empty list returns []", http.MethodGet, "/products", fakeProducts{}, http.StatusOK, 0},
		{"product by id", http.MethodGet, "/products/2", fakeProducts{items: two}, http.StatusOK, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			New(tt.products, fakeCarts{}, fakeOrders{}, quietLogger()).ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantCount < 0 {
				return
			}
			var got []api.Product
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("body is not a JSON array: %v (%s)", err, rec.Body.String())
			}
			if len(got) != tt.wantCount {
				t.Errorf("count = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}

// TestErrorResponsesShareOneShape is episode 3's incident as a test: whatever fails, and wherever it fails
// (our handler, the service, the generated binder, the router), the client gets the same JSON shape.
func TestErrorResponsesShareOneShape(t *testing.T) {
	boom := errors.New("connection refused: postgres://secret@db")
	wrappedNotFound := fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", service.ErrProductNotFound))
	tests := []struct {
		name       string
		method     string
		path       string
		products   fakeProducts
		wantStatus int
		wantCode   string
		wantField  string
	}{
		{"unknown id", http.MethodGet, "/products/99", fakeProducts{}, http.StatusNotFound, "product_not_found", ""},
		{"not found survives %w wrapping", http.MethodGet, "/products/1", fakeProducts{err: wrappedNotFound}, http.StatusNotFound, "product_not_found", ""},
		{"id is not a number (generated binder)", http.MethodGet, "/products/abc", fakeProducts{}, http.StatusBadRequest, "invalid_request", "id"},
		{"id is not positive (service validation)", http.MethodGet, "/products/0", fakeProducts{err: &service.ValidationError{Field: "id", Message: "must be a positive integer"}}, http.StatusBadRequest, "invalid_request", "id"},
		{"list fails", http.MethodGet, "/products", fakeProducts{err: boom}, http.StatusInternalServerError, "internal_error", ""},
		{"get fails", http.MethodGet, "/products/1", fakeProducts{err: boom}, http.StatusInternalServerError, "internal_error", ""},
		{"unknown route (router)", http.MethodGet, "/nope", fakeProducts{}, http.StatusNotFound, "route_not_found", ""},
		{"wrong method (router)", http.MethodPost, "/products", fakeProducts{}, http.StatusMethodNotAllowed, "method_not_allowed", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			New(tt.products, fakeCarts{}, fakeOrders{}, quietLogger()).ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Fatalf("Content-Type = %q, want application/json", ct)
			}
			if tt.wantStatus == http.StatusMethodNotAllowed && rec.Header().Get("Allow") != "GET" {
				t.Errorf("Allow = %q, want GET", rec.Header().Get("Allow"))
			}
			dec := json.NewDecoder(rec.Body)
			dec.DisallowUnknownFields() // no extra keys: the shape is exactly api.Error
			var got api.Error
			if err := dec.Decode(&got); err != nil {
				t.Fatalf("body is not an api.Error: %v", err)
			}
			if got.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", got.Code, tt.wantCode)
			}
			if got.Message == "" {
				t.Error("message is empty")
			}
			if strings.Contains(got.Message, "secret") || strings.Contains(got.Message, "postgres") {
				t.Errorf("message leaks internals: %q", got.Message)
			}
			gotField := ""
			if got.Field != nil {
				gotField = *got.Field
			}
			if gotField != tt.wantField {
				t.Errorf("field = %q, want %q", gotField, tt.wantField)
			}
		})
	}
}
