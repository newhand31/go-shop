//go:build integration

package repository_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/newhand31/go-shop/internal/repository"
	"github.com/newhand31/go-shop/internal/service"
)

func countRows(t *testing.T, pool *pgxpool.Pool, table string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM `+table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// failingCommit is a transaction whose Commit always fails after rolling back, the way a dropped connection or a
// deferred constraint would. Everything written before it must disappear, the outbox row included.
type failingCommit struct{ pgx.Tx }

func (f failingCommit) Commit(ctx context.Context) error {
	_ = f.Tx.Rollback(ctx)
	return errors.New("commit failed")
}

type failingCommitDB struct{ pool *pgxpool.Pool }

func (f failingCommitDB) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := f.pool.Begin(ctx)
	return failingCommit{tx}, err
}

func TestOutbox_PlaceWritesOneRowWithTheOrder(t *testing.T) {
	ctx := context.Background()
	pool := openMigrated(t)

	order, err := repository.NewOrders(pool).Place(ctx, []repository.OrderLine{{ProductID: 1, Quantity: 1}})
	if err != nil {
		t.Fatal(err)
	}
	var topic, payload string
	if err := pool.QueryRow(ctx, `SELECT topic, payload::text FROM outbox`).Scan(&topic, &payload); err != nil {
		t.Fatalf("want exactly one outbox row: %v", err)
	}
	if topic != repository.TopicOrderConfirmation {
		t.Fatalf("topic = %q", topic)
	}
	want := `{"order_id": ` + strconv.Itoa(order.ID) + `, "total_twd": ` + strconv.Itoa(order.TotalTWD) + `}`
	if payload != want {
		t.Fatalf("payload = %s, want %s", payload, want)
	}
}

// The reason the outbox exists: the order and its email promise live and die together.
func TestOutbox_RolledBackOrderLeavesNoOutboxRow(t *testing.T) {
	ctx := context.Background()
	pool := openMigrated(t)

	_, err := repository.NewOrders(failingCommitDB{pool}).Place(ctx, []repository.OrderLine{{ProductID: 1, Quantity: 1}})
	if err == nil {
		t.Fatal("want the commit error")
	}
	if n := countRows(t, pool, "outbox"); n != 0 {
		t.Fatalf("outbox has %d rows after a rolled-back order, want 0", n)
	}
	if n := countRows(t, pool, "orders"); n != 0 {
		t.Fatalf("orders has %d rows after a rolled-back order, want 0", n)
	}
	var stock int
	if err := pool.QueryRow(ctx, `SELECT stock FROM products WHERE id = 1`).Scan(&stock); err != nil || stock != 10 {
		t.Fatalf("stock = %d (err %v), want the seeded 10 back", stock, err)
	}
}

// flakyMailer fails its first failures calls, then works.
type flakyMailer struct {
	failures int32
	calls    atomic.Int32
}

func (m *flakyMailer) SendOrderConfirmation(context.Context, int, int) error {
	if m.calls.Add(1) <= m.failures {
		return errors.New("smtp down")
	}
	return nil
}

func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestOutbox_FailedSendIsRetriedUntilItWorks(t *testing.T) {
	ctx := context.Background()
	pool := openMigrated(t)
	if _, err := repository.NewOrders(pool).Place(ctx, []repository.OrderLine{{ProductID: 1, Quantity: 1}}); err != nil {
		t.Fatal(err)
	}

	mailer := &flakyMailer{failures: 2}
	d := service.NewOutboxDispatcher(repository.NewOutbox(pool), mailer, service.OutboxConfig{
		Interval: 5 * time.Millisecond,
		Backoff:  func(int) time.Duration { return 0 }, // retry on the next tick instead of waiting seconds
	}, quietLogger())

	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		d.Run(runCtx)
	}()

	deadline := time.Now().Add(20 * time.Second)
	for {
		var sent bool
		if err := pool.QueryRow(ctx, `SELECT sent_at IS NOT NULL FROM outbox`).Scan(&sent); err != nil {
			t.Fatal(err)
		}
		if sent {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("not sent after 20s; mailer was called %d times", mailer.calls.Load())
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	<-done // the worker has really stopped before we look at the final state

	var attempts int
	var lastError *string
	if err := pool.QueryRow(ctx, `SELECT attempts, last_error FROM outbox`).Scan(&attempts, &lastError); err != nil {
		t.Fatal(err)
	}
	if got := mailer.calls.Load(); got != 3 {
		t.Errorf("mailer called %d times, want 3 (two failures, one success)", got)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
	if lastError != nil {
		t.Errorf("last_error = %q, want it cleared after the success", *lastError)
	}

	// A sent row is never sent again.
	d.RunOnce(ctx)
	if got := mailer.calls.Load(); got != 3 {
		t.Errorf("mailer called %d times after the row was sent, want still 3", got)
	}
}

func TestOutbox_GivesUpAfterMaxAttemptsAndKeepsTheError(t *testing.T) {
	ctx := context.Background()
	pool := openMigrated(t)
	if _, err := repository.NewOrders(pool).Place(ctx, []repository.OrderLine{{ProductID: 1, Quantity: 1}}); err != nil {
		t.Fatal(err)
	}
	mailer := &flakyMailer{failures: 1000}
	d := service.NewOutboxDispatcher(repository.NewOutbox(pool), mailer, service.OutboxConfig{
		MaxAttempts: 3,
		Backoff:     func(int) time.Duration { return 0 },
	}, quietLogger())

	for range 6 {
		d.RunOnce(ctx)
	}
	if got := mailer.calls.Load(); got != 3 {
		t.Fatalf("mailer called %d times, want exactly MaxAttempts = 3", got)
	}
	var sent bool
	var lastError string
	if err := pool.QueryRow(ctx, `SELECT sent_at IS NOT NULL, last_error FROM outbox`).Scan(&sent, &lastError); err != nil {
		t.Fatal(err)
	}
	if sent || lastError != "smtp down" {
		t.Fatalf("sent = %v, last_error = %q; want an unsent row that remembers why", sent, lastError)
	}
}
