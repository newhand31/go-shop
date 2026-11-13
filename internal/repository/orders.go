package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"

	"github.com/newhand31/go-shop/internal/repository/dbgen"
)

// Sentinel errors the transaction can end with. The service translates them into its own.
var (
	ErrProductNotFound = errors.New("product not found")
	ErrOutOfStock      = errors.New("not enough stock")
)

// OrderLine is one product and how many of it to buy.
type OrderLine struct {
	ProductID int
	Quantity  int
}

// OrderedLine is a line as stored: the price is the one read under the lock, so it cannot change mid-order.
type OrderedLine struct {
	ProductID    int
	Quantity     int
	UnitPriceTWD int
}

// Order is a committed order.
type Order struct {
	ID       int
	TotalTWD int
	Lines    []OrderedLine
}

// TxBeginner is the one thing Orders needs from the pool. *pgxpool.Pool has it.
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Orders writes orders. Unlike the read-only repositories it needs a transaction, and Go has no [Transactional]:
// the boundary is three lines you write yourself, in the open.
type Orders struct {
	db TxBeginner
}

// NewOrders takes a pool (anything that can Begin).
func NewOrders(db TxBeginner) *Orders { return &Orders{db: db} }

// Place decrements stock and records the order, all or nothing.
//
// The stock check has to happen inside the transaction, while the row is locked: "read stock, decide, write" outside
// a lock is exactly the race that oversold the shop. That is why this one rule lives here, next to the lock.
func (r *Orders) Place(ctx context.Context, lines []OrderLine) (Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Order{}, fmt.Errorf("begin order: %w", err)
	}
	// If anything below returns early, the deferred Rollback undoes it. After a successful Commit it is a harmless no-op
	// (it returns pgx.ErrTxClosed, which we ignore). This pair, Begin + defer Rollback, is Go's whole "transaction scope".
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)

	// Lock rows in a fixed order (by product id). Two orders that want the same products in opposite order
	// would otherwise each hold one lock and wait for the other: a deadlock.
	sorted := append([]OrderLine(nil), lines...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ProductID < sorted[j].ProductID })

	order := Order{}
	for _, l := range sorted {
		p, err := q.LockProduct(ctx, int32(l.ProductID))
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, fmt.Errorf("product %d: %w", l.ProductID, ErrProductNotFound)
		}
		if err != nil {
			return Order{}, fmt.Errorf("lock product %d: %w", l.ProductID, err)
		}
		if int(p.Stock) < l.Quantity {
			return Order{}, fmt.Errorf("product %d has %d, want %d: %w", l.ProductID, p.Stock, l.Quantity, ErrOutOfStock)
		}
		if err := q.DecreaseStock(ctx, dbgen.DecreaseStockParams{ID: int32(l.ProductID), Stock: int32(l.Quantity)}); err != nil {
			return Order{}, fmt.Errorf("decrease stock of %d: %w", l.ProductID, err)
		}
		order.Lines = append(order.Lines, OrderedLine{ProductID: l.ProductID, Quantity: l.Quantity, UnitPriceTWD: int(p.PriceTwd)})
		order.TotalTWD += int(p.PriceTwd) * l.Quantity
	}

	id, err := q.InsertOrder(ctx, int32(order.TotalTWD))
	if err != nil {
		return Order{}, fmt.Errorf("insert order: %w", err)
	}
	for _, l := range order.Lines {
		err := q.InsertOrderLine(ctx, dbgen.InsertOrderLineParams{
			OrderID: id, ProductID: int32(l.ProductID), Quantity: int32(l.Quantity), UnitPriceTwd: int32(l.UnitPriceTWD),
		})
		if err != nil {
			return Order{}, fmt.Errorf("insert order line: %w", err)
		}
	}
	order.ID = int(id)

	// The confirmation email is a promise made in the same transaction as the order: the row below commits with the
	// order or rolls back with it. Sending the email here instead would be a network call inside a lock, and an order
	// that is committed but never mailed whenever the process dies in between.
	payload, err := confirmationPayload(order)
	if err != nil {
		return Order{}, fmt.Errorf("encode confirmation: %w", err)
	}
	if err := q.InsertOutbox(ctx, dbgen.InsertOutboxParams{Topic: TopicOrderConfirmation, Payload: payload}); err != nil {
		return Order{}, fmt.Errorf("insert outbox: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Order{}, fmt.Errorf("commit order: %w", err)
	}
	return order, nil
}
