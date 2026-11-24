package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/newhand31/go-shop/internal/repository"
)

// ErrOrderNotFound means no order has that id. The handler turns it into a 404.
var ErrOrderNotFound = errors.New("order not found")

// OrderStatus is where an order is in its life. An order is unpaid until ECPay says it was paid, or until the janitor
// gives its stock back, whichever comes first.
type OrderStatus string

// The three states an order can be in.
const (
	StatusUnpaid  OrderStatus = "unpaid"
	StatusPaid    OrderStatus = "paid"
	StatusExpired OrderStatus = "expired"
)

// OrderDetail is a stored order with its status.
type OrderDetail struct {
	Order
	Status OrderStatus
}

// OrderLookup is what reading an order needs from storage. repository.OrderReader fits without saying so.
type OrderLookup interface {
	Get(ctx context.Context, id int) (repository.OrderRecord, error)
}

// OrderReadService answers "what happened to this order?".
type OrderReadService struct {
	store OrderLookup
}

// NewOrderReadService gets its store injected by hand in main.
func NewOrderReadService(store OrderLookup) *OrderReadService { return &OrderReadService{store: store} }

// Get returns one order. A bad id is a *ValidationError, an unknown id wraps ErrOrderNotFound.
func (s *OrderReadService) Get(ctx context.Context, id int) (OrderDetail, error) {
	if id <= 0 {
		return OrderDetail{}, &ValidationError{Field: "orderId", Message: "must be a positive integer"}
	}
	rec, err := s.store.Get(ctx, id)
	switch {
	case errors.Is(err, repository.ErrOrderNotFound):
		return OrderDetail{}, fmt.Errorf("%w: %w", ErrOrderNotFound, err)
	case err != nil:
		return OrderDetail{}, fmt.Errorf("get order: %w", err)
	}
	out := OrderDetail{Order: Order{ID: rec.ID, TotalTWD: rec.TotalTWD}, Status: statusOf(rec)}
	for _, l := range rec.Lines {
		out.Lines = append(out.Lines, OrderLine{ProductID: l.ProductID, Quantity: l.Quantity, UnitPriceTWD: l.UnitPriceTWD})
	}
	return out, nil
}

// statusOf: paid wins. The janitor skips paid orders, but a callback that arrives after the janitor expired an order
// still marks it paid (the repository does not check expired_at), so both timestamps can be set. The money did arrive.
func statusOf(rec repository.OrderRecord) OrderStatus {
	switch {
	case rec.PaidAt != nil:
		return StatusPaid
	case rec.ExpiredAt != nil:
		return StatusExpired
	default:
		return StatusUnpaid
	}
}
