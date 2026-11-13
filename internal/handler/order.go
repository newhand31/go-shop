package handler

import (
	"context"
	"net/http"

	"github.com/newhand31/go-shop/internal/api"
	"github.com/newhand31/go-shop/internal/service"
)

// Orders is the service capability the checkout route needs, defined by the consumer.
type Orders interface {
	Checkout(ctx context.Context, cartID string) (service.Order, error)
}

// Checkout buys the visitor's cart. The cart id is the session id, as for the cart routes.
func (s *server) Checkout(w http.ResponseWriter, r *http.Request) {
	id := SessionID(r.Context())
	if id == "" {
		writeError(w, s.logger, errNoSession)
		return
	}
	order, err := s.orders.Checkout(r.Context(), id)
	if err != nil {
		writeError(w, s.logger, err)
		return
	}
	out := api.Order{Id: order.ID, TotalTwd: order.TotalTWD, Lines: make([]api.OrderLine, 0, len(order.Lines))}
	for _, l := range order.Lines {
		out.Lines = append(out.Lines, api.OrderLine{ProductId: l.ProductID, Quantity: l.Quantity, UnitPriceTwd: l.UnitPriceTWD})
	}
	writeJSON(w, s.logger, http.StatusCreated, out)
}
