package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/newhand31/go-shop/internal/repository/dbgen"
)

// ErrPaymentMismatch means the payment names no such order, or names a different amount than the order's total.
var ErrPaymentMismatch = errors.New("payment does not match an order")

// Payment is one payment as reported by the payment provider.
type Payment struct {
	OrderID   int
	TradeNo   string // the provider's transaction number: the idempotency key
	AmountTWD int
}

// Payments records payments.
type Payments struct {
	db TxBeginner
}

// NewPayments takes a pool (anything that can Begin).
func NewPayments(db TxBeginner) *Payments { return &Payments{db: db} }

// pgForeignKeyViolation is SQLSTATE 23503: the order id does not exist.
const pgForeignKeyViolation = "23503"

// Record stores the payment and marks its order paid, once.
//
// It reports recorded=false when this TradeNo was already recorded: a resend of a callback we have already handled.
// That is not an error. The UNIQUE constraint on trade_no decides, so two resends arriving at the same instant still
// produce exactly one row; no "SELECT first, then INSERT" race.
func (r *Payments) Record(ctx context.Context, p Payment) (recorded bool, err error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin payment: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := dbgen.New(tx)

	n, err := q.InsertPayment(ctx, dbgen.InsertPaymentParams{OrderID: int64(p.OrderID), TradeNo: p.TradeNo, AmountTwd: int32(p.AmountTWD)})
	var pgErr *pgconn.PgError
	switch {
	case errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation:
		return false, fmt.Errorf("order %d: %w", p.OrderID, ErrPaymentMismatch)
	case err != nil:
		return false, fmt.Errorf("insert payment: %w", err)
	case n == 0:
		return false, nil // already recorded; the first call did the work
	}

	n, err = q.MarkOrderPaid(ctx, dbgen.MarkOrderPaidParams{ID: int64(p.OrderID), TotalTwd: int32(p.AmountTWD)})
	if err != nil {
		return false, fmt.Errorf("mark order %d paid: %w", p.OrderID, err)
	}
	if n == 0 {
		// Wrong amount. Returning rolls back the payment row too, so nothing half-recorded is left behind.
		return false, fmt.Errorf("order %d, amount %d: %w", p.OrderID, p.AmountTWD, ErrPaymentMismatch)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit payment: %w", err)
	}
	return true, nil
}
