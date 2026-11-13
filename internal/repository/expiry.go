package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/newhand31/go-shop/internal/repository/dbgen"
)

// Expiry gives stock back from orders nobody paid for.
type Expiry struct {
	db TxBeginner
}

// NewExpiry takes a pool (anything that can Begin).
func NewExpiry(db TxBeginner) *Expiry { return &Expiry{db: db} }

// ReleaseOne releases one unpaid order created before cutoff: its stock goes back to the shelf and the order is stamped
// expired, in one transaction. It reports ok=false when there is nothing to release.
//
// One order per call keeps every transaction short. The caller loops, and can stop between orders when its time is up.
func (e *Expiry) ReleaseOne(ctx context.Context, cutoff time.Time) (orderID int, ok bool, err error) {
	tx, err := e.db.Begin(ctx)
	if err != nil {
		return 0, false, fmt.Errorf("begin release: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	q := dbgen.New(tx)

	id, err := q.ClaimExpiredOrder(ctx, pgtype.Timestamptz{Time: cutoff, Valid: true})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("claim expired order: %w", err)
	}
	lines, err := q.ListOrderLines(ctx, id)
	if err != nil {
		return 0, false, fmt.Errorf("list lines of order %d: %w", id, err)
	}
	for _, l := range lines {
		if err := q.IncreaseStock(ctx, dbgen.IncreaseStockParams{ID: l.ProductID, Stock: l.Quantity}); err != nil {
			return 0, false, fmt.Errorf("return stock of product %d: %w", l.ProductID, err)
		}
	}
	if err := q.MarkOrderExpired(ctx, id); err != nil {
		return 0, false, fmt.Errorf("mark order %d expired: %w", id, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, false, fmt.Errorf("commit release: %w", err)
	}
	return int(id), true, nil
}
