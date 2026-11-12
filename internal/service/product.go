// Package service 放業務邏輯。它不碰 http.Request。
package service

import (
	"context"
	"fmt"

	"github.com/newhand31/go-shop/internal/repository"
)

// ProductStore 是 service 需要的資料存取，由使用端（這裡）定義，只列用到的方法。
type ProductStore interface {
	List(ctx context.Context) ([]repository.Product, error)
}

// Product 是 service 對外的商品型別，handler 只認這個。
type Product struct {
	ID       int
	Name     string
	PriceTWD int
}

// ProductService 負責商品相關業務。
type ProductService struct {
	store ProductStore
}

// NewProductService 由 main 手動注入 store。
func NewProductService(store ProductStore) *ProductService {
	return &ProductService{store: store}
}

// List 回傳所有商品。
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
