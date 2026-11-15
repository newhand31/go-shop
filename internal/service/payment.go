package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/newhand31/go-shop/internal/payment/ecpay"
	"github.com/newhand31/go-shop/internal/repository"
)

// Errors the handler maps to a response for ECPay.
var (
	// ErrInvalidCallback means the body was forged, tampered with or malformed. Do not retry it.
	ErrInvalidCallback = errors.New("invalid payment callback")
	// ErrPaymentMismatch means the callback is genuine but names an unknown order or a wrong amount.
	ErrPaymentMismatch = errors.New("payment does not match an order")
)

// PaymentStore is what the callback needs from storage. repository.Payments fits without saying so.
type PaymentStore interface {
	Record(ctx context.Context, p repository.Payment) (recorded bool, err error)
}

// CallbackVerifier checks and parses a callback. *ecpay.Verifier fits.
type CallbackVerifier interface {
	Verify(params url.Values) (ecpay.Callback, error)
}

// PaymentService turns ECPay's callback into a recorded payment.
type PaymentService struct {
	verifier CallbackVerifier
	store    PaymentStore
	logger   *slog.Logger
}

// NewPaymentService gets its dependencies injected by hand in main.
func NewPaymentService(verifier CallbackVerifier, store PaymentStore, logger *slog.Logger) *PaymentService {
	return &PaymentService{verifier: verifier, store: store, logger: logger}
}

// HandleECPayCallback verifies the signature, then records the payment at most once.
//
// ECPay resends a callback until it gets "1|OK" (and also when our reply was lost), so the same callback arrives more
// than once. A duplicate returns nil like the first: from ECPay's side both succeeded. A failed payment (RtnCode other
// than 1) is acknowledged too, since resending will not change its outcome.
func (s *PaymentService) HandleECPayCallback(ctx context.Context, params url.Values) error {
	cb, err := s.verifier.Verify(params)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidCallback, err)
	}
	if !cb.Paid {
		s.logger.InfoContext(ctx, "payment not completed", "trade_no", cb.TradeNo, "merchant_trade_no", cb.MerchantTradeNo, "msg", cb.RtnMsg)
		return nil
	}
	orderID, err := ecpay.ParseOrderRef(cb.MerchantTradeNo)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidCallback, err)
	}

	recorded, err := s.store.Record(ctx, repository.Payment{OrderID: orderID, TradeNo: cb.TradeNo, AmountTWD: cb.AmountTWD})
	switch {
	case errors.Is(err, repository.ErrPaymentMismatch):
		return fmt.Errorf("%w: %w", ErrPaymentMismatch, err)
	case err != nil:
		return fmt.Errorf("handle callback: %w", err)
	case !recorded:
		s.logger.InfoContext(ctx, "duplicate payment callback ignored", "trade_no", cb.TradeNo, "order", orderID)
	default:
		s.logger.InfoContext(ctx, "payment recorded", "trade_no", cb.TradeNo, "order", orderID, "amount_twd", cb.AmountTWD)
	}
	return nil
}
