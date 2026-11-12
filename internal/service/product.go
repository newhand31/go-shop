// Package service holds the business logic. It never touches http.Request.
package service

import (
	"context"
	"fmt"

	"github.com/newhand31/go-shop/internal/repository"
)

// ProductStore is the data access the service needs, defined by the consumer (here) and listing only the methods it uses.
type ProductStore interface {
	List(ctx context.Context) ([]repository.Product, error)
}

// Product is the product type the service exposes; the handler only knows this one.
type Product struct {
	ID       int
	Name     string
	PriceTWD int
}

// ProductService handles product business logic.
type ProductService struct {
	store ProductStore
}

// NewProductService gets its store injected by hand in main.
func NewProductService(store ProductStore) *ProductService {
	return &ProductService{store: store}
}

// List returns all products.
func (s *ProductService) List(ctx context.Context) ([]Product, error) {
	rows, err := s.store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	out := make([]Product, 0, len(rows))
	for _, r := range rows {
		out = append(out, Product{ID: r.ID, Name: r.Name, PriceTWD: r.PriceTWD})
	}
	return out, nil
}
