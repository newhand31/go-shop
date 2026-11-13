package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/newhand31/go-shop/internal/service"
)

type fakePayments struct{ err error }

func (f fakePayments) HandleECPayCallback(context.Context, url.Values) error { return f.err }

func callbackHandler(p Payments) http.Handler {
	return WithPaymentCallback(New(fakeProducts{}, fakeCarts{}, fakeOrders{}, quietLogger()), p, quietLogger())
}

func postForm(h http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, ECPayCallbackPath, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestECPayCallbackRoute(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{"accepted", nil, http.StatusOK, "1|OK"},
		{"forged", fmt.Errorf("x: %w", service.ErrInvalidCallback), http.StatusBadRequest, "0|rejected"},
		{"wrong amount", fmt.Errorf("x: %w", service.ErrPaymentMismatch), http.StatusBadRequest, "0|rejected"},
		// A body that says "1|OK" for a storage failure would make ECPay stop resending a payment we never recorded.
		{"our failure is not acknowledged", errors.New("db down: 10.0.0.5"), http.StatusInternalServerError, "0|error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := postForm(callbackHandler(fakePayments{err: tt.err}), "TradeNo=T1")
			if rec.Code != tt.wantStatus || rec.Body.String() != tt.wantBody {
				t.Errorf("got %d %q, want %d %q", rec.Code, rec.Body.String(), tt.wantStatus, tt.wantBody)
			}
		})
	}
}

func TestWithPaymentCallback_LeavesTheAPIReachable(t *testing.T) {
	rec := httptest.NewRecorder()
	callbackHandler(fakePayments{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("/healthz status = %d, want 200", rec.Code)
	}
}

func TestECPayCallbackRoute_OversizedBodyIsRejected(t *testing.T) {
	rec := postForm(callbackHandler(fakePayments{}), "a="+strings.Repeat("x", maxCallbackBytes+10))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}
