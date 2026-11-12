// Package repository is the data access layer. Episode 1 uses hard-coded in-memory data; episode 2 switches to PostgreSQL.
package repository

import "context"

// Product is the concrete type the repository returns.
type Product struct {
	ID       int
	Name     string
	PriceTWD int
}

// MemoryProducts keeps products hard-coded in memory.
type MemoryProducts struct {
	items []Product
}

// NewMemoryProducts creates a repository with three built-in products.
func NewMemoryProducts() *MemoryProducts {
	return &MemoryProducts{items: []Product{
		{ID: 1, Name: "機械式鍵盤", PriceTWD: 2490},
		{ID: 2, Name: "人體工學滑鼠", PriceTWD: 1290},
		{ID: 3, Name: "27 吋螢幕", PriceTWD: 5990},
	}}
}

// List returns a copy of all products, so callers cannot change the internal data.
func (r *MemoryProducts) List(_ context.Context) ([]Product, error) {
	out := make([]Product, len(r.items))
	copy(out, r.items)
	return out, nil
}
