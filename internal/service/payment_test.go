package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/url"
	"testing"

	"github.com/newhand31/go-shop/internal/payment/ecpay"
	"github.com/newhand31/go-shop/internal/repository"
)

// fakePayments remembers what it was asked to record and answers with a canned result.
type fakePayments struct {
	recorded bool
	err      error
	got      []repository.Payment
}

func (f *fakePayments) Record(_ context.Context, p repository.Payment) (bool, error) {
	f.got = append(f.got, p)
	return f.recorded, f.err
}

func signedCallback(v *ecpay.Verifier, ref, tradeNo, amount, rtnCode string) url.Values {
	return v.Sign(url.Values{
		"MerchantID": {ecpay.SandboxMerchantID}, "MerchantTradeNo": {ref}, "TradeNo": {tradeNo},
		"TradeAmt": {amount}, "RtnCode": {rtnCode}, "RtnMsg": {"x"},
	})
}

// errStorage is a marker for "any error that is not one of the callback errors".
var errStorage = errors.New("storage")

func TestHandleECPayCallback(t *testing.T) {
	v := ecpay.NewVerifier(ecpay.SandboxMerchantID, ecpay.SandboxHashKey, ecpay.SandboxHashIV)
	forged := signedCallback(v, "GS42", "T1", "1290", "1")
	forged.Set("TradeAmt", "1")
	want42 := &repository.Payment{OrderID: 42, TradeNo: "T1", AmountTWD: 1290}

	tests := []struct {
		name       string
		params     url.Values
		store      *fakePayments
		wantErr    error
		wantRecord *repository.Payment // nil: the store must not be called
	}{
		{"first delivery is recorded", signedCallback(v, "GS42", "T1", "1290", "1"), &fakePayments{recorded: true}, nil, want42},
		{"a resend is not an error", signedCallback(v, "GS42", "T1", "1290", "1"), &fakePayments{recorded: false}, nil, want42},
		{"forged amount never reaches the store", forged, &fakePayments{}, ErrInvalidCallback, nil},
		{"unknown order reference", signedCallback(v, "ZZ9", "T1", "1290", "1"), &fakePayments{}, ErrInvalidCallback, nil},
		{"failed payment is acknowledged, not recorded", signedCallback(v, "GS42", "T1", "1290", "10100058"), &fakePayments{}, nil, nil},
		{"wrong amount for the order", signedCallback(v, "GS42", "T1", "1290", "1"), &fakePayments{err: repository.ErrPaymentMismatch}, ErrPaymentMismatch, want42},
		{"storage failure is not swallowed", signedCallback(v, "GS42", "T1", "1290", "1"), &fakePayments{err: errors.New("db down")}, errStorage, want42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewPaymentService(v, tt.store, slog.New(slog.NewTextHandler(io.Discard, nil)))
			err := svc.HandleECPayCallback(context.Background(), tt.params)
			switch {
			case tt.wantErr == nil && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.wantErr == errStorage:
				if err == nil || errors.Is(err, ErrInvalidCallback) || errors.Is(err, ErrPaymentMismatch) {
					t.Fatalf("err = %v, want a plain storage error", err)
				}
			case tt.wantErr != nil && !errors.Is(err, tt.wantErr):
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			switch {
			case tt.wantRecord == nil && len(tt.store.got) != 0:
				t.Fatalf("store was called with %+v, want no call", tt.store.got)
			case tt.wantRecord != nil && (len(tt.store.got) != 1 || tt.store.got[0] != *tt.wantRecord):
				t.Fatalf("store got %+v, want %+v", tt.store.got, *tt.wantRecord)
			}
		})
	}
}
