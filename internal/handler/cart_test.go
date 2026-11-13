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

// fakeCarts is a Carts with canned answers; it records the cart id it was asked about.
type fakeCarts struct {
	cart   service.Cart
	err    error
	gotID  *string
	gotQty *int
}

func (f fakeCarts) record(id string) {
	if f.gotID != nil {
		*f.gotID = id
	}
}

func (f fakeCarts) Get(_ context.Context, id string) (service.Cart, error) {
	f.record(id)
	return f.cart, f.err
}

func (f fakeCarts) SetItem(_ context.Context, id string, _, qty int) (service.Cart, error) {
	f.record(id)
	if f.gotQty != nil {
		*f.gotQty = qty
	}
	return f.cart, f.err
}

func (f fakeCarts) RemoveItem(_ context.Context, id string, _ int) (service.Cart, error) {
	f.record(id)
	return f.cart, f.err
}

// withSession pretends the Session middleware ran, so cart routes see a session id.
func withSession(id string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, id)))
		})
	}
}

func TestCartRoutes(t *testing.T) {
	oneLine := service.Cart{Items: []service.CartItem{{ProductID: 1, Name: "keyboard", PriceTWD: 1500, Quantity: 2}}, TotalTWD: 3000}
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		carts      fakeCarts
		wantStatus int
		wantCode   string // error code, "" for success
	}{
		{"get cart", http.MethodGet, "/cart", "", fakeCarts{cart: oneLine}, http.StatusOK, ""},
		{"put item", http.MethodPut, "/cart/items/1", `{"quantity":2}`, fakeCarts{cart: oneLine}, http.StatusOK, ""},
		{"delete item", http.MethodDelete, "/cart/items/1", "", fakeCarts{cart: oneLine}, http.StatusOK, ""},
		{"put with bad json", http.MethodPut, "/cart/items/1", `{"quantity":`, fakeCarts{}, http.StatusBadRequest, "invalid_request"},
		{"put with unknown field", http.MethodPut, "/cart/items/1", `{"quantity":1,"price":0}`, fakeCarts{}, http.StatusBadRequest, "invalid_request"},
		{"put with non-numeric product id", http.MethodPut, "/cart/items/abc", `{"quantity":1}`, fakeCarts{}, http.StatusBadRequest, "invalid_request"},
		{"put unknown product", http.MethodPut, "/cart/items/9", `{"quantity":1}`, fakeCarts{err: fmt.Errorf("x: %w", service.ErrProductNotFound)}, http.StatusNotFound, "product_not_found"},
		{"put bad quantity", http.MethodPut, "/cart/items/1", `{"quantity":0}`, fakeCarts{err: &service.ValidationError{Field: "quantity", Message: "must be between 1 and 99"}}, http.StatusBadRequest, "invalid_request"},
		{"store failure is a 500", http.MethodGet, "/cart", "", fakeCarts{err: errors.New("redis down: tcp 10.0.0.5:6379")}, http.StatusInternalServerError, "internal_error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			New(fakeProducts{}, tt.carts, quietLogger(), withSession("s1")).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, tt.wantStatus, rec.Body)
			}
			if tt.wantCode != "" {
				if e := decodeError(t, rec); e.Code != tt.wantCode {
					t.Fatalf("code = %q, want %q", e.Code, tt.wantCode)
				}
				if strings.Contains(rec.Body.String(), "10.0.0.5") {
					t.Fatal("internal detail leaked to the client")
				}
				return
			}
			var got api.Cart
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			if got.TotalTwd != 3000 || len(got.Items) != 1 || got.Items[0].Name != "keyboard" {
				t.Fatalf("cart = %+v", got)
			}
		})
	}
}

func TestCartUsesSessionIDAsCartID(t *testing.T) {
	var gotID string
	var gotQty int
	carts := fakeCarts{gotID: &gotID, gotQty: &gotQty}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/cart/items/1", strings.NewReader(`{"quantity":7}`))
	New(fakeProducts{}, carts, quietLogger(), withSession("session-42")).ServeHTTP(rec, req)

	if gotID != "session-42" || gotQty != 7 {
		t.Fatalf("service got cart id %q quantity %d, want session-42 and 7", gotID, gotQty)
	}
}

// Without the Session middleware there is no cart id. That is a wiring bug: 500, never a shared "" cart.
func TestCartWithoutSessionIsAServerError(t *testing.T) {
	rec := httptest.NewRecorder()
	New(fakeProducts{}, fakeCarts{}, quietLogger()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/cart", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
