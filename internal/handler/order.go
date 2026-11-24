package handler

import (
	"context"
	"net/http"

	"github.com/newhand31/go-shop/internal/api"
	"github.com/newhand31/go-shop/internal/service"
)

// Orders is what the order routes need from the service layer, defined by the consumer: place an order, read one back,
// and start paying one.
type Orders interface {
	Checkout(ctx context.Context, cartID string) (service.Order, error)
	Get(ctx context.Context, id int) (service.OrderDetail, error)
	StartPayment(ctx context.Context, orderID int) (service.PaymentForm, error)
}

// Checkout buys the visitor's cart. The cart id is the session id, as for the cart routes.
func (s *server) Checkout(w http.ResponseWriter, r *http.Request) {
	id := SessionID(r.Context())
	if id == "" {
		writeError(w, r, s.logger, errNoSession)
		return
	}
	order, err := s.orders.Checkout(r.Context(), id)
	if err != nil {
		writeError(w, r, s.logger, err)
		return
	}
	out := api.Order{Id: order.ID, TotalTwd: order.TotalTWD, Lines: make([]api.OrderLine, 0, len(order.Lines))}
	for _, l := range order.Lines {
		out.Lines = append(out.Lines, api.OrderLine{ProductId: l.ProductID, Quantity: l.Quantity, UnitPriceTwd: l.UnitPriceTWD})
	}
	writeJSON(w, s.logger, http.StatusCreated, out)
}

// GetOrder serves GET /orders/{orderId}. There are no accounts yet, so any visitor who knows an order id can read it.
func (s *server) GetOrder(w http.ResponseWriter, r *http.Request, orderID int) {
	order, err := s.orders.Get(r.Context(), orderID)
	if err != nil {
		writeError(w, r, s.logger, err)
		return
	}
	out := api.OrderDetail{Id: order.ID, TotalTwd: order.TotalTWD, Status: api.OrderDetailStatus(order.Status), Lines: make([]api.OrderLine, 0, len(order.Lines))}
	for _, l := range order.Lines {
		out.Lines = append(out.Lines, api.OrderLine{ProductId: l.ProductID, Quantity: l.Quantity, UnitPriceTwd: l.UnitPriceTWD})
	}
	writeJSON(w, s.logger, http.StatusOK, out)
}

// StartPayment serves POST /orders/{orderId}/payment: the browser gets a signed form and posts it to ECPay itself.
func (s *server) StartPayment(w http.ResponseWriter, r *http.Request, orderID int) {
	form, err := s.orders.StartPayment(r.Context(), orderID)
	if err != nil {
		writeError(w, r, s.logger, err)
		return
	}
	writeJSON(w, s.logger, http.StatusOK, api.PaymentForm{OrderId: form.OrderID, ActionUrl: form.ActionURL, Method: api.POST, Fields: form.Fields})
}
