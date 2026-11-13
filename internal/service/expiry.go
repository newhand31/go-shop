package service

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// ExpiryStore is what the janitor needs from storage. repository.Expiry fits without saying so.
type ExpiryStore interface {
	// ReleaseOne frees one unpaid order created before cutoff; ok=false means there was none left.
	ReleaseOne(ctx context.Context, cutoff time.Time) (orderID int, ok bool, err error)
}

// ExpiryConfig tunes the janitor. Zero fields get defaults. Time is a field you can replace: a test passes a tiny
// Interval and a Now that jumps an hour ahead, instead of waiting for either.
type ExpiryConfig struct {
	Interval     time.Duration    // how often to sweep (default 1 min)
	OrderTTL     time.Duration    // how long an order may stay unpaid (default 15 min)
	RoundTimeout time.Duration    // the budget of one sweep (default 30 s)
	Now          func() time.Time // the clock (default time.Now)
}

func (c ExpiryConfig) withDefaults() ExpiryConfig {
	if c.Interval <= 0 {
		c.Interval = time.Minute
	}
	if c.OrderTTL <= 0 {
		c.OrderTTL = 15 * time.Minute
	}
	if c.RoundTimeout <= 0 {
		c.RoundTimeout = 30 * time.Second
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	return c
}

// ExpiryJanitor releases unpaid orders so they stop holding stock.
type ExpiryJanitor struct {
	store  ExpiryStore
	cfg    ExpiryConfig
	logger *slog.Logger
}

// NewExpiryJanitor gets its dependencies injected by hand in main.
func NewExpiryJanitor(store ExpiryStore, cfg ExpiryConfig, logger *slog.Logger) *ExpiryJanitor {
	return &ExpiryJanitor{store: store, cfg: cfg.withDefaults(), logger: logger}
}

// Run sweeps every Interval until ctx is cancelled, then returns. Like the outbox worker it blocks: the caller starts
// the goroutine and owns stopping it.
func (j *ExpiryJanitor) Run(ctx context.Context) {
	ticker := time.NewTicker(j.cfg.Interval)
	defer ticker.Stop()
	for {
		j.Sweep(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Sweep is one round. It gets its own deadline: a sweep stuck on a lock or a slow database is cancelled after
// RoundTimeout, and the next tick starts clean. It returns how many orders it released.
//
// context.WithTimeout is the CancellationTokenSource(TimeSpan) of Go, except the deadline rides inside the context and
// every call below (down to the SQL driver) sees it.
func (j *ExpiryJanitor) Sweep(ctx context.Context) int {
	if ctx.Err() != nil {
		return 0
	}
	roundCtx, cancel := context.WithTimeout(ctx, j.cfg.RoundTimeout)
	defer cancel()

	cutoff := j.cfg.Now().Add(-j.cfg.OrderTTL)
	released := 0
	for {
		id, ok, err := j.store.ReleaseOne(roundCtx, cutoff)
		switch {
		case errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil:
			j.logger.Warn("expiry sweep ran out of time, next round continues", "released", released, "timeout", j.cfg.RoundTimeout)
			return released
		case err != nil:
			if ctx.Err() == nil {
				j.logger.Error("expiry sweep failed", "err", err, "released", released)
			}
			return released
		case !ok:
			if released > 0 {
				j.logger.Info("released unpaid orders", "count", released)
			}
			return released
		}
		j.logger.Info("released unpaid order", "order_id", id)
		released++
	}
}
