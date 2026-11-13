package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"

	"github.com/newhand31/go-shop/internal/repository"
)

// ErrOutOfStock means the shop has fewer of some product than the order asks for. The handler turns it into a 409.
var ErrOutOfStock = errors.New("out of stock")

// OrderStore is what checkout needs from storage: place the whole order or none of it.
// repository.Orders fits without saying so; so does the plain-SELECT version the concurrency test uses to show the bug.
type OrderStore interface {
	Place(ctx context.Context, lines []repository.OrderLine) (repository.Order, error)
}

// CheckoutCart is the part of a cart checkout touches: read it, then empty it. repository.Carts fits this one too.
type CheckoutCart interface {
	Items(ctx context.Context, cartID string) (map[int]int, error)
	RemoveItem(ctx context.Context, cartID string, productID int) error
}

// OrderLine is one line of a placed order.
type OrderLine struct {
	ProductID    int
	Quantity     int
	UnitPriceTWD int
}

// Order is what the handler shows after a successful checkout.
type Order struct {
	ID       int
	TotalTWD int
	Lines    []OrderLine
}

// OrderService turns a cart into an order.
type OrderService struct {
	store  OrderStore
	carts  CheckoutCart
	logger *slog.Logger
}

// NewOrderService gets its dependencies injected by hand in main.
func NewOrderService(store OrderStore, carts CheckoutCart, logger *slog.Logger) *OrderService {
	return &OrderService{store: store, carts: carts, logger: logger}
}

// Checkout buys everything in the cart. An empty cart is a *ValidationError; not enough stock wraps ErrOutOfStock
// and leaves both the stock and the cart untouched.
func (s *OrderService) Checkout(ctx context.Context, cartID string) (Order, error) {
	items, err := s.carts.Items(ctx, cartID)
	if err != nil {
		return Order{}, fmt.Errorf("checkout: read cart: %w", err)
	}
	if len(items) == 0 {
		return Order{}, &ValidationError{Field: "cart", Message: "is empty"}
	}
	lines := make([]repository.OrderLine, 0, len(items))
	for id, qty := range items {
		lines = append(lines, repository.OrderLine{ProductID: id, Quantity: qty})
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].ProductID < lines[j].ProductID })

	placed, err := s.store.Place(ctx, lines)
	switch {
	case errors.Is(err, repository.ErrOutOfStock):
		return Order{}, fmt.Errorf("checkout: %w: %w", ErrOutOfStock, err)
	case errors.Is(err, repository.ErrProductNotFound):
		return Order{}, fmt.Errorf("checkout: %w: %w", ErrProductNotFound, err)
	case err != nil:
		return Order{}, fmt.Errorf("checkout: %w", err)
	}

	// The order is committed. Emptying the cart is a courtesy: if Redis hiccups here we must not report a failed
	// order for money-moving work that already happened, so log and move on. (Making this idempotent is out of scope.)
	for _, l := range lines {
		if err := s.carts.RemoveItem(ctx, cartID, l.ProductID); err != nil {
			s.logger.Warn("order placed but cart not emptied", "order", placed.ID, "product", l.ProductID, "err", err)
		}
	}

	out := Order{ID: placed.ID, TotalTWD: placed.TotalTWD}
	for _, l := range placed.Lines {
		out.Lines = append(out.Lines, OrderLine{ProductID: l.ProductID, Quantity: l.Quantity, UnitPriceTWD: l.UnitPriceTWD})
	}
	return out, nil
}
