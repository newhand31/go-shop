package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/newhand31/go-shop/internal/payment/ecpay"
	"github.com/newhand31/go-shop/internal/repository"
)

type fakeLookup struct {
	rec repository.OrderRecord
	err error
}

func (f fakeLookup) Get(context.Context, int) (repository.OrderRecord, error) { return f.rec, f.err }

func TestOrderReadServiceGet(t *testing.T) {
	now := time.Now()
	lines := []repository.OrderedLine{{ProductID: 3, Quantity: 2, UnitPriceTWD: 100}}
	tests := []struct {
		name       string
		id         int
		store      fakeLookup
		wantStatus OrderStatus
		wantErr    error
		wantField  string
	}{
		{"unpaid", 1, fakeLookup{rec: repository.OrderRecord{ID: 1, TotalTWD: 200, Lines: lines}}, StatusUnpaid, nil, ""},
		{"paid", 1, fakeLookup{rec: repository.OrderRecord{ID: 1, TotalTWD: 200, Lines: lines, PaidAt: &now}}, StatusPaid, nil, ""},
		{"expired", 1, fakeLookup{rec: repository.OrderRecord{ID: 1, TotalTWD: 200, Lines: lines, ExpiredAt: &now}}, StatusExpired, nil, ""},
		{"paid after expiry is still paid", 1, fakeLookup{rec: repository.OrderRecord{ID: 1, Lines: lines, PaidAt: &now, ExpiredAt: &now}}, StatusPaid, nil, ""},
		{"unknown order", 9, fakeLookup{err: repository.ErrOrderNotFound}, "", ErrOrderNotFound, ""},
		{"zero id is invalid", 0, fakeLookup{}, "", nil, "orderId"},
		{"store failure passes through", 1, fakeLookup{err: errors.New("db down")}, "", errors.New("db down"), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewOrderReadService(tt.store).Get(context.Background(), tt.id)
			var ve *ValidationError
			switch {
			case tt.wantField != "":
				if !errors.As(err, &ve) || ve.Field != tt.wantField {
					t.Fatalf("err = %v, want ValidationError on %q", err, tt.wantField)
				}
			case tt.wantErr != nil:
				if err == nil || (errors.Is(tt.wantErr, ErrOrderNotFound) && !errors.Is(err, ErrOrderNotFound)) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
			default:
				if err != nil {
					t.Fatal(err)
				}
				if got.Status != tt.wantStatus || len(got.Lines) != 1 || got.Lines[0].UnitPriceTWD != 100 {
					t.Errorf("got %+v, want status %s with one line", got, tt.wantStatus)
				}
			}
		})
	}
}

func TestPaymentStarterStartPayment(t *testing.T) {
	now := time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)
	verifier := ecpay.NewVerifier(ecpay.SandboxMerchantID, ecpay.SandboxHashKey, ecpay.SandboxHashIV)
	paid := now
	tests := []struct {
		name    string
		store   fakeLookup
		wantErr error
	}{
		{"unpaid order gets a form", fakeLookup{rec: repository.OrderRecord{ID: 12, TotalTWD: 5990}}, nil},
		{"paid order cannot be paid again", fakeLookup{rec: repository.OrderRecord{ID: 12, TotalTWD: 5990, PaidAt: &paid}}, ErrOrderNotPayable},
		{"expired order cannot be paid", fakeLookup{rec: repository.OrderRecord{ID: 12, TotalTWD: 5990, ExpiredAt: &paid}}, ErrOrderNotPayable},
		{"unknown order", fakeLookup{err: repository.ErrOrderNotFound}, ErrOrderNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewPaymentStarter(tt.store, verifier, "https://shop.example/payments/ecpay/callback")
			s.now = func() time.Time { return now }
			form, err := s.StartPayment(context.Background(), 12)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if form.OrderID != 12 || form.ActionURL != verifier.CheckoutURL() {
				t.Errorf("form = %+v", form)
			}
			want := map[string]string{"MerchantTradeNo": ecpay.AttemptRef(12, time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)), "TotalAmount": "5990", "ReturnURL": "https://shop.example/payments/ecpay/callback", "MerchantTradeDate": "2026/11/24 18:00:00"}
			for k, v := range want {
				if form.Fields[k] != v {
					t.Errorf("field %s = %q, want %q", k, form.Fields[k], v)
				}
			}
			// The form must carry a signature ECPay's rule accepts: recompute it from the other fields.
			fields := map[string][]string{}
			for k, v := range form.Fields {
				fields[k] = []string{v}
			}
			if got, want := form.Fields[ecpay.CheckMacField], verifier.CheckMacValue(fields); got != want {
				t.Errorf("CheckMacValue = %q, recomputed %q", got, want)
			}
		})
	}
}
