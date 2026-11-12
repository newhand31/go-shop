// Package repository is the data access layer: sqlc-generated queries plus a thin wrapper. No business rules here.
package repository

import (
	"context"
	"fmt"

	"github.com/newhand31/go-shop/internal/repository/dbgen"
)

// Product is the concrete type the repository returns.
type Product struct {
	ID       int
	Name     string
	PriceTWD int
}

// Products reads products from PostgreSQL through sqlc-generated code.
type Products struct {
	q *dbgen.Queries
}

// NewProducts takes anything sqlc can run queries on: a *pgxpool.Pool today, a pgx.Tx when transactions arrive.
func NewProducts(conn dbgen.DBTX) *Products {
	return &Products{q: dbgen.New(conn)}
}

// List returns all products ordered by id.
func (r *Products) List(ctx context.Context) ([]Product, error) {
	rows, err := r.q.ListProducts(ctx)
	if err != nil {
		return nil, fmt.Errorf("query products: %w", err)
	}
	out := make([]Product, 0, len(rows))
	for _, row := range rows {
		out = append(out, Product{ID: int(row.ID), Name: row.Name, PriceTWD: int(row.PriceTwd)})
	}
	return out, nil
}
