package service

import (
	"context"
	"fmt"
	"sort"
)

// Quantity limits for one cart line.
const (
	minQuantity = 1
	maxQuantity = 99
)

// CartStore is what the cart logic needs from storage. It is declared here, next to its only user, and lists
// only the three methods used. The Redis implementation lives in another package and never mentions this interface.
type CartStore interface {
	Items(ctx context.Context, cartID string) (map[int]int, error)
	SetItem(ctx context.Context, cartID string, productID, quantity int) error
	RemoveItem(ctx context.Context, cartID string, productID int) error
}

// Catalog is the slice of the product service the cart needs. *ProductService fits it without being told.
type Catalog interface {
	List(ctx context.Context) ([]Product, error)
}

// CartItem is one line of a cart, with the product's current name and price.
type CartItem struct {
	ProductID int
	Name      string
	PriceTWD  int
	Quantity  int
}

// Cart is what the handler shows.
type Cart struct {
	Items    []CartItem
	TotalTWD int
}

// CartService holds the cart rules: which products may be added, how many, and how a total is computed.
type CartService struct {
	store   CartStore
	catalog Catalog
}

// NewCartService gets its dependencies injected by hand in main.
func NewCartService(store CartStore, catalog Catalog) *CartService {
	return &CartService{store: store, catalog: catalog}
}

// Get returns the cart. Lines whose product no longer exists are left out rather than failing the whole cart.
func (s *CartService) Get(ctx context.Context, cartID string) (Cart, error) {
	items, err := s.store.Items(ctx, cartID)
	if err != nil {
		return Cart{}, fmt.Errorf("get cart: %w", err)
	}
	products, err := s.products(ctx)
	if err != nil {
		return Cart{}, fmt.Errorf("get cart: %w", err)
	}

	cart := Cart{Items: make([]CartItem, 0, len(items))}
	for id, qty := range items {
		p, ok := products[id]
		if !ok {
			continue
		}
		cart.Items = append(cart.Items, CartItem{ProductID: id, Name: p.Name, PriceTWD: p.PriceTWD, Quantity: qty})
		cart.TotalTWD += p.PriceTWD * qty
	}
	// Map iteration order is random in Go; sort so the same cart always renders the same way.
	sort.Slice(cart.Items, func(i, j int) bool { return cart.Items[i].ProductID < cart.Items[j].ProductID })
	return cart, nil
}

// SetItem sets the quantity of a product (1 to 99) and returns the new cart.
// A bad quantity is a *ValidationError; an unknown product wraps ErrProductNotFound.
func (s *CartService) SetItem(ctx context.Context, cartID string, productID, quantity int) (Cart, error) {
	if productID <= 0 {
		return Cart{}, &ValidationError{Field: "productId", Message: "must be a positive integer"}
	}
	if quantity < minQuantity || quantity > maxQuantity {
		return Cart{}, &ValidationError{Field: "quantity", Message: fmt.Sprintf("must be between %d and %d", minQuantity, maxQuantity)}
	}
	products, err := s.products(ctx)
	if err != nil {
		return Cart{}, fmt.Errorf("set cart item: %w", err)
	}
	if _, ok := products[productID]; !ok {
		return Cart{}, fmt.Errorf("set cart item %d: %w", productID, ErrProductNotFound)
	}
	if err := s.store.SetItem(ctx, cartID, productID, quantity); err != nil {
		return Cart{}, fmt.Errorf("set cart item: %w", err)
	}
	return s.Get(ctx, cartID)
}

// RemoveItem removes a product from the cart and returns the new cart.
func (s *CartService) RemoveItem(ctx context.Context, cartID string, productID int) (Cart, error) {
	if productID <= 0 {
		return Cart{}, &ValidationError{Field: "productId", Message: "must be a positive integer"}
	}
	if err := s.store.RemoveItem(ctx, cartID, productID); err != nil {
		return Cart{}, fmt.Errorf("remove cart item: %w", err)
	}
	return s.Get(ctx, cartID)
}

func (s *CartService) products(ctx context.Context) (map[int]Product, error) {
	all, err := s.catalog.List(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[int]Product, len(all))
	for _, p := range all {
		byID[p.ID] = p
	}
	return byID, nil
}
