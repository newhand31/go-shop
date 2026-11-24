package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/newhand31/go-shop/internal/payment/ecpay"
)

// ErrOrderNotPayable means the order is already paid or has expired, so starting a payment makes no sense.
var ErrOrderNotPayable = errors.New("order is not payable")

// CheckoutSigner builds ECPay's checkout form. *ecpay.Verifier fits.
type CheckoutSigner interface {
	CheckoutURL() string
	CheckoutFields(r ecpay.CheckoutRequest) url.Values
}

// PaymentForm is what the browser needs to send the visitor to ECPay: POST Fields to ActionURL.
type PaymentForm struct {
	OrderID   int
	ActionURL string
	Fields    map[string]string
}

// PaymentStarter starts the ECPay payment of an order.
type PaymentStarter struct {
	orders    OrderLookup
	signer    CheckoutSigner
	returnURL string
	now       func() time.Time
}

// NewPaymentStarter gets its dependencies injected by hand in main. returnURL is the public address of the callback
// route: ECPay's server must be able to reach it.
func NewPaymentStarter(orders OrderLookup, signer CheckoutSigner, returnURL string) *PaymentStarter {
	return &PaymentStarter{orders: orders, signer: signer, returnURL: returnURL, now: time.Now}
}

// StartPayment returns the signed form for one unpaid order. It stores nothing: the order row already says what is owed, and
// the callback is what records the payment. Calling it twice therefore returns the same fields, apart from the date.
func (s *PaymentStarter) StartPayment(ctx context.Context, orderID int) (PaymentForm, error) {
	order, err := NewOrderReadService(s.orders).Get(ctx, orderID)
	if err != nil {
		return PaymentForm{}, err
	}
	if order.Status != StatusUnpaid {
		return PaymentForm{}, fmt.Errorf("order %d is %s: %w", orderID, order.Status, ErrOrderNotPayable)
	}
	fields := s.signer.CheckoutFields(ecpay.CheckoutRequest{
		OrderID:     order.ID,
		AmountTWD:   order.TotalTWD,
		ItemName:    fmt.Sprintf("go-shop order %d", order.ID),
		Description: "go-shop",
		ReturnURL:   s.returnURL,
		Now:         s.now(),
	})
	form := PaymentForm{OrderID: order.ID, ActionURL: s.signer.CheckoutURL(), Fields: make(map[string]string, len(fields))}
	for k := range fields {
		form.Fields[k] = fields.Get(k)
	}
	return form, nil
}
