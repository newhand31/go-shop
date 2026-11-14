package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/newhand31/go-shop/internal/repository"
)

// ErrBusy means the shop is full right now: the queue in front of the database has no free slot. The handler turns it
// into a 503 with Retry-After. The request never reached the database; that is the point.
var ErrBusy = errors.New("shop is busy")

// StockGate is the cheap check in front of the database: take units from a counter, give them back.
// repository.Reservations fits without saying so.
type StockGate interface {
	Reserve(ctx context.Context, productID, qty int) (ok bool, err error)
	Release(ctx context.Context, productID, qty int) error
}

// FlashConfig sizes the two limits of a flash sale. Zero fields get defaults.
//
// Workers is how many orders may be in the database at once. Keep it well below the pool size: the pool also serves
// the outbox worker, the janitor and every read. QueueSize is how many buyers may wait behind the workers; the next
// one is turned away. Together they are the shop's whole capacity: Workers + QueueSize requests, not one more.
type FlashConfig struct {
	Workers   int
	QueueSize int
}

func (c FlashConfig) withDefaults() FlashConfig {
	if c.Workers <= 0 {
		c.Workers = 4
	}
	if c.QueueSize <= 0 {
		c.QueueSize = 64
	}
	return c
}

// flashJob is one buyer waiting in the queue. claimed settles who owns it: the worker that picks it up, or the buyer
// who gives up first. Whoever wins the swap is the only one allowed to settle the reservation, so it is released once.
type flashJob struct {
	ctx       context.Context
	productID int
	qty       int
	claimed   atomic.Bool
	done      chan flashResult // buffered: the worker never blocks on a buyer who left
}

type flashResult struct {
	order Order
	err   error
}

func (j *flashJob) claim() bool { return j.claimed.CompareAndSwap(false, true) }

// FlashSale sells a limited product under a stampede. The path is: Redis counter (cheap, atomic) → bounded queue (a
// channel with a fixed buffer) → a few workers → one database transaction each. Every step can say no, and each no
// is cheaper than the step after it.
type FlashSale struct {
	gate    StockGate
	orders  OrderStore
	cfg     FlashConfig
	logger  *slog.Logger
	queue   chan *flashJob
	stopped chan struct{} // closed when Run returns
}

// NewFlashSale gets its dependencies injected by hand in main.
func NewFlashSale(gate StockGate, orders OrderStore, cfg FlashConfig, logger *slog.Logger) *FlashSale {
	cfg = cfg.withDefaults()
	return &FlashSale{
		gate: gate, orders: orders, cfg: cfg, logger: logger,
		queue:   make(chan *flashJob, cfg.QueueSize),
		stopped: make(chan struct{}),
	}
}

// Buy tries to buy qty units of one product. It never waits for a free slot: no room means ErrBusy, right now.
//
//   - sold out at the counter        → ErrOutOfStock (the database is not touched)
//   - queue full                     → ErrBusy       (the reservation is given back, the database is not touched)
//   - the database says out of stock → ErrOutOfStock (the reservation is given back)
func (f *FlashSale) Buy(ctx context.Context, productID, qty int) (Order, error) {
	if qty <= 0 {
		return Order{}, &ValidationError{Field: "quantity", Message: "must be positive"}
	}
	ok, err := f.gate.Reserve(ctx, productID, qty)
	if err != nil {
		return Order{}, fmt.Errorf("flash sale: reserve: %w", err)
	}
	if !ok {
		return Order{}, ErrOutOfStock
	}

	job := &flashJob{ctx: ctx, productID: productID, qty: qty, done: make(chan flashResult, 1)}
	// The back-pressure line. A plain `f.queue <- job` would block when the buffer is full, and a blocked request
	// holds its goroutine, its connection and its memory: the queue would just move to somewhere we cannot see.
	// A select with a default never blocks, so a full queue becomes an answer instead of a wait.
	select {
	case f.queue <- job:
	default:
		f.release(job)
		return Order{}, ErrBusy
	}

	var abandon error
	select {
	case res := <-job.done:
		return res.order, res.err
	case <-ctx.Done():
		abandon = ctx.Err()
	case <-f.stopped:
		abandon = ErrBusy // nobody is going to pick this up any more
	}
	// We stopped waiting. If no worker has started on the job, it is ours: give the unit back and leave the job in
	// the queue as a corpse, which the worker will skip. If a worker already claimed it, it is mid-transaction: wait.
	if job.claim() {
		f.release(job)
		return Order{}, abandon
	}
	res := <-job.done
	return res.order, res.err
}

// Run starts the workers and blocks until ctx is cancelled and every worker has returned. Like the outbox worker and
// the janitor, the caller starts the goroutine and owns stopping it.
func (f *FlashSale) Run(ctx context.Context) {
	defer close(f.stopped)
	var wg sync.WaitGroup
	for range f.cfg.Workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job := <-f.queue:
					f.handle(job)
				}
			}
		}()
	}
	wg.Wait()
}

func (f *FlashSale) handle(job *flashJob) {
	if !job.claim() {
		return // the buyer left while waiting; their unit is already back
	}
	placed, err := f.orders.Place(job.ctx, []repository.OrderLine{{ProductID: job.productID, Quantity: job.qty}})
	if err != nil {
		f.release(job)
		switch {
		case errors.Is(err, repository.ErrOutOfStock):
			err = ErrOutOfStock
		case errors.Is(err, repository.ErrProductNotFound):
			err = ErrProductNotFound
		default:
			err = fmt.Errorf("flash sale: place order: %w", err)
		}
		job.done <- flashResult{err: err}
		return
	}
	out := Order{ID: placed.ID, TotalTWD: placed.TotalTWD}
	for _, l := range placed.Lines {
		out.Lines = append(out.Lines, OrderLine{ProductID: l.ProductID, Quantity: l.Quantity, UnitPriceTWD: l.UnitPriceTWD})
	}
	job.done <- flashResult{order: out}
}

// release is the compensation. The buyer's own context may be the reason we are here (they hung up), so it gets a
// fresh, short one: a cancelled context would fail the very call that must not fail.
func (f *FlashSale) release(job *flashJob) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(job.ctx), 2*time.Second)
	defer cancel()
	if err := f.gate.Release(ctx, job.productID, job.qty); err != nil {
		f.logger.Error("flash sale: could not give a reserved unit back", "product", job.productID, "qty", job.qty, "err", err)
	}
}
