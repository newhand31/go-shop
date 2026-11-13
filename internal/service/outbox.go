package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/newhand31/go-shop/internal/repository"
)

// Mailer is what the outbox worker needs from an email system. It is defined here, where it is used, not next to
// any SMTP code: the real sender, a logger, and the fake in the tests all fit by having this one method.
type Mailer interface {
	SendOrderConfirmation(ctx context.Context, orderID, totalTWD int) error
}

// OutboxStore is what the worker needs from storage. repository.Outbox fits without saying so.
type OutboxStore interface {
	Dispatch(
		ctx context.Context,
		limit, maxAttempts int,
		backoff func(attempts int) time.Duration,
		send func(ctx context.Context, m repository.OutboxMessage) error,
	) (int, error)
}

// OutboxConfig tunes the worker. Zero fields get defaults, so production passes OutboxConfig{} and a test passes
// a tiny Interval instead of sleeping.
type OutboxConfig struct {
	Interval    time.Duration                    // how often to look for due rows (default 2s)
	BatchSize   int                              // rows per look (default 20)
	MaxAttempts int                              // after this many failures the row is left for a human (default 8)
	Backoff     func(attempts int) time.Duration // wait before the next try (default 1s, doubling, capped at 5 min)
}

func (c OutboxConfig) withDefaults() OutboxConfig {
	if c.Interval <= 0 {
		c.Interval = 2 * time.Second
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 20
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 8
	}
	if c.Backoff == nil {
		c.Backoff = exponentialBackoff
	}
	return c
}

// exponentialBackoff is 1s, 2s, 4s, ... capped at five minutes.
func exponentialBackoff(attempts int) time.Duration {
	const maxWait = 5 * time.Minute
	if attempts < 1 {
		attempts = 1
	}
	if attempts > 9 {
		return maxWait
	}
	d := time.Second << (attempts - 1)
	return min(d, maxWait)
}

// OutboxDispatcher turns outbox rows into emails, in the background.
type OutboxDispatcher struct {
	store  OutboxStore
	mailer Mailer
	cfg    OutboxConfig
	logger *slog.Logger
}

// NewOutboxDispatcher gets its dependencies injected by hand in main.
func NewOutboxDispatcher(store OutboxStore, mailer Mailer, cfg OutboxConfig, logger *slog.Logger) *OutboxDispatcher {
	return &OutboxDispatcher{store: store, mailer: mailer, cfg: cfg.withDefaults(), logger: logger}
}

// Run looks for due rows every Interval until ctx is cancelled, then returns. It blocks, so the caller decides how
// to run it: "go d.Run(ctx)" plus something to wait on. A goroutine has no Task to await and no owner; whoever
// starts it is responsible for stopping it and for knowing it has stopped.
func (d *OutboxDispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(d.cfg.Interval)
	defer ticker.Stop()
	for {
		d.RunOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// RunOnce handles one batch. Errors are logged, never fatal: a database blip must not kill the worker for good.
func (d *OutboxDispatcher) RunOnce(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	sent, err := d.store.Dispatch(ctx, d.cfg.BatchSize, d.cfg.MaxAttempts, d.cfg.Backoff, d.send)
	if err != nil && ctx.Err() == nil {
		d.logger.Error("outbox batch failed", "err", err)
		return
	}
	if sent > 0 {
		d.logger.Info("outbox sent", "count", sent)
	}
}

// send does one row. An error here is not lost: the store keeps it and tries again later.
func (d *OutboxDispatcher) send(ctx context.Context, m repository.OutboxMessage) error {
	err := d.deliver(ctx, m)
	if err != nil {
		d.logger.Warn("outbox send failed, will retry", "id", m.ID, "topic", m.Topic, "attempt", m.Attempts+1, "err", err)
	}
	return err
}

func (d *OutboxDispatcher) deliver(ctx context.Context, m repository.OutboxMessage) error {
	switch m.Topic {
	case repository.TopicOrderConfirmation:
		var c repository.OrderConfirmation
		if err := json.Unmarshal(m.Payload, &c); err != nil {
			return fmt.Errorf("decode payload: %w", err)
		}
		return d.mailer.SendOrderConfirmation(ctx, c.OrderID, c.TotalTWD)
	default:
		return fmt.Errorf("unknown topic %q", m.Topic)
	}
}
