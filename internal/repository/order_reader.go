package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/newhand31/go-shop/internal/repository/dbgen"
)

// ErrOrderNotFound means no order has that id.
var ErrOrderNotFound = errors.New("order not found")

// OrderRecord is an order as stored, with the two timestamps that decide its status. A nil time means "not yet".
type OrderRecord struct {
	ID        int
	TotalTWD  int
	Lines     []OrderedLine
	PaidAt    *time.Time
	ExpiredAt *time.Time
}

// OrderReader reads orders. It only reads, so a plain pool is enough: no transaction.
type OrderReader struct {
	db dbgen.DBTX
}

// NewOrderReader takes a pool (anything that can run a query).
func NewOrderReader(db dbgen.DBTX) *OrderReader { return &OrderReader{db: db} }

// Get returns one order and its lines, or an error wrapping ErrOrderNotFound.
func (r *OrderReader) Get(ctx context.Context, id int) (OrderRecord, error) {
	q := dbgen.New(r.db)
	row, err := q.GetOrder(ctx, int64(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return OrderRecord{}, fmt.Errorf("order %d: %w", id, ErrOrderNotFound)
	}
	if err != nil {
		return OrderRecord{}, fmt.Errorf("get order %d: %w", id, err)
	}
	lines, err := q.ListOrderLinesWithPrice(ctx, int64(id))
	if err != nil {
		return OrderRecord{}, fmt.Errorf("list lines of order %d: %w", id, err)
	}
	rec := OrderRecord{ID: int(row.ID), TotalTWD: int(row.TotalTwd), PaidAt: timePtr(row.PaidAt.Valid, row.PaidAt.Time), ExpiredAt: timePtr(row.ExpiredAt.Valid, row.ExpiredAt.Time)}
	for _, l := range lines {
		rec.Lines = append(rec.Lines, OrderedLine{ProductID: int(l.ProductID), Quantity: int(l.Quantity), UnitPriceTWD: int(l.UnitPriceTwd)})
	}
	return rec, nil
}

func timePtr(valid bool, t time.Time) *time.Time {
	if !valid {
		return nil
	}
	return &t
}
