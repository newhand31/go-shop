package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/newhand31/go-shop/internal/api"
	"github.com/newhand31/go-shop/internal/service"
)

// fakeOrders is an Orders with a canned answer; it records the cart id it was asked about.
type fakeOrders struct {
	order  service.Order
	detail service.OrderDetail
	form   service.PaymentForm
	err    error
	gotID  *string
}

func (f fakeOrders) Get(context.Context, int) (service.OrderDetail, error) { return f.detail, f.err }

func (f fakeOrders) StartPayment(context.Context, int) (service.PaymentForm, error) {
	return f.form, f.err
}

func (f fakeOrders) Checkout(_ context.Context, id string) (service.Order, error) {
	if f.gotID != nil {
		*f.gotID = id
	}
	return f.order, f.err
}

func TestCheckoutRoute(t *testing.T) {
	placed := service.Order{ID: 7, TotalTWD: 5990, Lines: []service.OrderLine{{ProductID: 3, Quantity: 1, UnitPriceTWD: 5990}}}
	tests := []struct {
		name       string
		orders     fakeOrders
		wantStatus int
		wantCode   string
	}{
		{"placed", fakeOrders{order: placed}, http.StatusCreated, ""},
		{"out of stock is a 409", fakeOrders{err: fmt.Errorf("checkout: %w", service.ErrOutOfStock)}, http.StatusConflict, "out_of_stock"},
		{"empty cart is a 400", fakeOrders{err: &service.ValidationError{Field: "cart", Message: "is empty"}}, http.StatusBadRequest, "invalid_request"},
		{"vanished product is a 404", fakeOrders{err: fmt.Errorf("x: %w", service.ErrProductNotFound)}, http.StatusNotFound, "product_not_found"},
		{"database failure is a 500", fakeOrders{err: errors.New("commit: tcp 10.0.0.9:5432 reset")}, http.StatusInternalServerError, "internal_error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
			New(fakeProducts{}, fakeCarts{}, tt.orders, quietLogger(), withSession("s1")).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, tt.wantStatus, rec.Body)
			}
			if tt.wantCode != "" {
				if e := decodeError(t, rec); e.Code != tt.wantCode {
					t.Fatalf("code = %q, want %q", e.Code, tt.wantCode)
				}
				if strings.Contains(rec.Body.String(), "10.0.0.9") {
					t.Fatal("internal detail leaked to the client")
				}
				return
			}
			var got api.Order
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			if got.Id != 7 || got.TotalTwd != 5990 || len(got.Lines) != 1 || got.Lines[0].ProductId != 3 {
				t.Fatalf("order = %+v", got)
			}
		})
	}
}

func TestCheckoutUsesSessionIDAsCartID(t *testing.T) {
	var gotID string
	rec := httptest.NewRecorder()
	New(fakeProducts{}, fakeCarts{}, fakeOrders{gotID: &gotID}, quietLogger(), withSession("session-42")).
		ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/checkout", nil))
	if gotID != "session-42" {
		t.Fatalf("service got cart id %q, want session-42", gotID)
	}
}

func TestCheckoutWithoutSessionIsAServerError(t *testing.T) {
	rec := httptest.NewRecorder()
	New(fakeProducts{}, fakeCarts{}, fakeOrders{}, quietLogger()).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/checkout", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestOrderRoutes(t *testing.T) {
	detail := service.OrderDetail{Order: service.Order{ID: 7, TotalTWD: 5990, Lines: []service.OrderLine{{ProductID: 3, Quantity: 1, UnitPriceTWD: 5990}}}, Status: service.StatusPaid}
	form := service.PaymentForm{OrderID: 7, ActionURL: "https://pay.example/checkout", Fields: map[string]string{"MerchantTradeNo": "GS7", "CheckMacValue": "ABC"}}
	notFound := fmt.Errorf("get order: %w", service.ErrOrderNotFound)
	tests := []struct {
		name       string
		method     string
		path       string
		orders     fakeOrders
		wantStatus int
		wantCode   string
	}{
		{"read a paid order", http.MethodGet, "/orders/7", fakeOrders{detail: detail}, http.StatusOK, ""},
		{"read an unknown order", http.MethodGet, "/orders/99", fakeOrders{err: notFound}, http.StatusNotFound, "order_not_found"},
		{"read with a text id", http.MethodGet, "/orders/abc", fakeOrders{}, http.StatusBadRequest, "invalid_request"},
		{"read with id zero", http.MethodGet, "/orders/0", fakeOrders{err: &service.ValidationError{Field: "orderId", Message: "must be a positive integer"}}, http.StatusBadRequest, "invalid_request"},
		{"read, database failure", http.MethodGet, "/orders/7", fakeOrders{err: errors.New("tcp 10.0.0.9:5432 reset")}, http.StatusInternalServerError, "internal_error"},
		{"start payment", http.MethodPost, "/orders/7/payment", fakeOrders{form: form}, http.StatusOK, ""},
		{"start payment, unknown order", http.MethodPost, "/orders/99/payment", fakeOrders{err: notFound}, http.StatusNotFound, "order_not_found"},
		{"start payment, already paid", http.MethodPost, "/orders/7/payment", fakeOrders{err: fmt.Errorf("order 7 is paid: %w", service.ErrOrderNotPayable)}, http.StatusConflict, "order_not_payable"},
		{"start payment with GET is a 405", http.MethodGet, "/orders/7/payment", fakeOrders{}, http.StatusMethodNotAllowed, "method_not_allowed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			New(fakeProducts{}, fakeCarts{}, tt.orders, quietLogger(), withSession("s1")).ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, tt.wantStatus, rec.Body)
			}
			if tt.wantCode != "" {
				var e api.Error
				if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil || e.Code != tt.wantCode {
					t.Fatalf("error = %+v (%v), want code %q", e, err, tt.wantCode)
				}
				return
			}
			switch tt.method {
			case http.MethodGet:
				var got api.OrderDetail
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				if got.Id != 7 || got.Status != api.Paid || len(got.Lines) != 1 || got.Lines[0].UnitPriceTwd != 5990 {
					t.Errorf("order = %+v", got)
				}
			default:
				var got api.PaymentForm
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				if got.OrderId != 7 || got.Method != api.POST || got.ActionUrl != form.ActionURL || got.Fields["CheckMacValue"] != "ABC" {
					t.Errorf("form = %+v", got)
				}
			}
		})
	}
}
