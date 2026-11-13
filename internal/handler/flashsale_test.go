package handler

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/newhand31/go-shop/internal/service"
)

type stubSales struct {
	order service.Order
	err   error
}

func (s stubSales) Buy(context.Context, int, int) (service.Order, error) { return s.order, s.err }

func TestFlashSaleBuy_StatusMapping(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tests := []struct {
		name       string
		path       string
		sales      stubSales
		wantStatus int
		wantRetry  string
	}{
		{"bought", "/flash-sales/3/orders", stubSales{order: service.Order{ID: 7, TotalTWD: 5990}}, http.StatusCreated, ""},
		{"sold out", "/flash-sales/3/orders", stubSales{err: service.ErrOutOfStock}, http.StatusConflict, ""},
		{"queue full is 503 with Retry-After", "/flash-sales/3/orders", stubSales{err: service.ErrBusy}, http.StatusServiceUnavailable, "1"},
		{"bad product id", "/flash-sales/abc/orders", stubSales{}, http.StatusBadRequest, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.NotFoundHandler()
			h := WithFlashSale(next, tt.sales, logger)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, tt.path, nil))
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if got := rec.Header().Get("Retry-After"); got != tt.wantRetry {
				t.Errorf("Retry-After = %q, want %q", got, tt.wantRetry)
			}
		})
	}
}
