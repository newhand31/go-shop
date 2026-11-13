package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/newhand31/go-shop/internal/repository/dbgen"
)

// TopicOrderConfirmation marks an outbox row that asks for an order-confirmation email.
const TopicOrderConfirmation = "order_confirmation"

// OrderConfirmation is the payload of a TopicOrderConfirmation row. It is written inside the order's transaction,
// so it can only exist for an order that really was committed.
type OrderConfirmation struct {
	OrderID  int `json:"order_id"`
	TotalTWD int `json:"total_twd"`
}

// OutboxMessage is one claimed row.
type OutboxMessage struct {
	ID       int64
	Topic    string
	Payload  []byte
	Attempts int // failed or sent attempts so far, before this one
}

// Outbox reads the outbox table for the background worker. Writing happens in Orders.Place, not here.
type Outbox struct {
	db TxBeginner
}

// NewOutbox takes a pool (anything that can Begin).
func NewOutbox(db TxBeginner) *Outbox { return &Outbox{db: db} }

// Dispatch claims up to limit due rows, calls send for each, and records the outcome, all in one transaction.
//
// A row whose send returns nil is marked sent. A row whose send fails stays unsent: attempts goes up, the error is
// kept, and the row becomes due again after backoff(attempts). Rows that already failed maxAttempts times are left
// alone for a human. It returns how many rows were sent.
//
// The rows stay locked while send runs (FOR UPDATE SKIP LOCKED), so two workers never send the same row. The price is
// that the transaction is open during the network call, which is fine for a handful of rows and the first thing to
// change if send gets slow.
func (o *Outbox) Dispatch(
	ctx context.Context,
	limit, maxAttempts int,
	backoff func(attempts int) time.Duration,
	send func(ctx context.Context, m OutboxMessage) error,
) (int, error) {
	tx, err := o.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin outbox batch: %w", err)
	}
	// Not ctx: if the worker is being shut down, ctx is already cancelled and a Rollback on it would not run.
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	q := dbgen.New(tx)
	// Recording an outcome must survive a shutdown: the email is already out, so the "sent" mark has to land.
	recordCtx := context.WithoutCancel(ctx)

	rows, err := q.ClaimOutbox(ctx, dbgen.ClaimOutboxParams{Attempts: int32(maxAttempts), Limit: int32(limit)})
	if err != nil {
		return 0, fmt.Errorf("claim outbox: %w", err)
	}
	sent := 0
	for _, row := range rows {
		if ctx.Err() != nil {
			break // shutting down: leave the rest alone; what is already marked still gets committed
		}
		m := OutboxMessage{ID: row.ID, Topic: row.Topic, Payload: row.Payload, Attempts: int(row.Attempts)}
		if sendErr := send(ctx, m); sendErr != nil {
			if ctx.Err() != nil {
				break // cancelled mid-send: that is shutdown, not a failed attempt
			}
			err := q.MarkOutboxFailed(recordCtx, dbgen.MarkOutboxFailedParams{
				ID: row.ID, LastError: textOf(sendErr.Error()), BackoffSecs: backoff(m.Attempts + 1).Seconds(),
			})
			if err != nil {
				return sent, fmt.Errorf("mark outbox %d failed: %w", row.ID, err)
			}
			continue
		}
		if err := q.MarkOutboxSent(recordCtx, row.ID); err != nil {
			return sent, fmt.Errorf("mark outbox %d sent: %w", row.ID, err)
		}
		sent++
	}
	if err := tx.Commit(recordCtx); err != nil {
		return 0, fmt.Errorf("commit outbox batch: %w", err)
	}
	return sent, nil
}

// confirmationPayload is what Place writes for a new order.
func confirmationPayload(o Order) ([]byte, error) {
	return json.Marshal(OrderConfirmation{OrderID: o.ID, TotalTWD: o.TotalTWD})
}

func textOf(s string) pgtype.Text { return pgtype.Text{String: s, Valid: true} }
