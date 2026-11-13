package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/newhand31/go-shop/internal/api"
	"github.com/newhand31/go-shop/internal/service"
)

// Carts is the service capability the cart routes need, defined by the consumer.
// The cart is identified by the session id the Session middleware put on the context.
type Carts interface {
	Get(ctx context.Context, cartID string) (service.Cart, error)
	SetItem(ctx context.Context, cartID string, productID, quantity int) (service.Cart, error)
	RemoveItem(ctx context.Context, cartID string, productID int) (service.Cart, error)
}

// errNoSession means a cart route ran without the Session middleware: a wiring bug, not a client mistake.
var errNoSession = errors.New("no session on the request context; is the Session middleware installed?")

func (s *server) GetCart(w http.ResponseWriter, r *http.Request) {
	id := SessionID(r.Context())
	if id == "" {
		writeError(w, s.logger, errNoSession)
		return
	}
	cart, err := s.carts.Get(r.Context(), id)
	s.writeCart(w, cart, err)
}

func (s *server) PutCartItem(w http.ResponseWriter, r *http.Request, productID int) {
	id := SessionID(r.Context())
	if id == "" {
		writeError(w, s.logger, errNoSession)
		return
	}
	var in api.CartItemInput
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil || dec.More() {
		writeError(w, s.logger, &service.ValidationError{Field: "body", Message: "must be a JSON object like {\"quantity\": 2}"})
		return
	}
	cart, err := s.carts.SetItem(r.Context(), id, productID, in.Quantity)
	s.writeCart(w, cart, err)
}

func (s *server) DeleteCartItem(w http.ResponseWriter, r *http.Request, productID int) {
	id := SessionID(r.Context())
	if id == "" {
		writeError(w, s.logger, errNoSession)
		return
	}
	cart, err := s.carts.RemoveItem(r.Context(), id, productID)
	s.writeCart(w, cart, err)
}

func (s *server) writeCart(w http.ResponseWriter, cart service.Cart, err error) {
	if err != nil {
		writeError(w, s.logger, err)
		return
	}
	out := api.Cart{Items: make([]api.CartItem, 0, len(cart.Items)), TotalTwd: cart.TotalTWD}
	for _, it := range cart.Items {
		out.Items = append(out.Items, api.CartItem{ProductId: it.ProductID, Name: it.Name, PriceTwd: it.PriceTWD, Quantity: it.Quantity})
	}
	writeJSON(w, s.logger, http.StatusOK, out)
}
