//go:build integration

package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/newhand31/go-shop/internal/platform/postgres"
	"github.com/newhand31/go-shop/internal/platform/redis"
	"github.com/newhand31/go-shop/internal/repository"
	"github.com/newhand31/go-shop/internal/service"
)

const monitorID = 3 // the seeded 27-inch monitor

// stack is the real thing: PostgreSQL and Redis in throwaway containers, a pool with a small ceiling, and the monitor
// restocked to `stock` units in both places.
type stack struct {
	pool *pgxpool.Pool
	res  *repository.Reservations
}

func startStack(t *testing.T, maxConns int32, stock int) stack {
	t.Helper()
	ctx := context.Background()

	pg, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("goshop"), tcpostgres.WithUsername("goshop"), tcpostgres.WithPassword("goshop"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	testcontainers.CleanupContainer(t, pg)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	dbURL, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	rd, err := tcredis.Run(ctx, "redis:7-alpine")
	testcontainers.CleanupContainer(t, rd)
	if err != nil {
		t.Fatalf("start redis: %v", err)
	}
	redisURL, err := rd.ConnectionString(ctx)
	if err != nil {
		t.Fatal(err)
	}

	pool, err := postgres.OpenWithMaxConns(ctx, dbURL, maxConns)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := postgres.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE products SET stock = $2 WHERE id = $1`, monitorID, stock); err != nil {
		t.Fatal(err)
	}
	rdb, err := redis.Open(ctx, redisURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	res := repository.NewReservations(rdb)
	if _, err := res.Open(ctx, monitorID, stock); err != nil {
		t.Fatal(err)
	}
	return stack{pool: pool, res: res}
}

// countingStore counts the calls that reach the database layer.
type countingStore struct {
	inner service.OrderStore
	calls atomic.Int32
}

func (c *countingStore) Place(ctx context.Context, lines []repository.OrderLine) (repository.Order, error) {
	c.calls.Add(1)
	return c.inner.Place(ctx, lines)
}

func (s stack) startSale(t *testing.T, cfg service.FlashConfig) (*service.FlashSale, *countingStore) {
	t.Helper()
	store := &countingStore{inner: repository.NewOrders(s.pool)}
	sale := service.NewFlashSale(s.res, store, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); sale.Run(ctx) }()
	t.Cleanup(func() { cancel(); <-done })
	return sale, store
}

type tally struct{ bought, soldOut, busy atomic.Int32 }

// stampede sends `buyers` Buy calls at the same instant and sorts the answers.
func stampede(t *testing.T, sale *service.FlashSale, buyers int) *tally {
	t.Helper()
	var out tally
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range buyers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := sale.Buy(context.Background(), monitorID, 1)
			switch {
			case err == nil:
				out.bought.Add(1)
			case errors.Is(err, service.ErrOutOfStock):
				out.soldOut.Add(1)
			case errors.Is(err, service.ErrBusy):
				out.busy.Add(1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	return &out
}

func orderCount(t *testing.T, s stack) (orders, stock int) {
	t.Helper()
	ctx := context.Background()
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM orders`).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if err := s.pool.QueryRow(ctx, `SELECT stock FROM products WHERE id = $1`, monitorID).Scan(&stock); err != nil {
		t.Fatal(err)
	}
	return orders, stock
}

// Proof 1: 5 units, 100 buyers at once. Exactly 5 orders, the other 95 hear "sold out", the database stock lands on 0
// (never negative), and the database saw 5 order attempts, not 100.
func TestFlashSale_Integration_NeverOversells(t *testing.T) {
	const stock, buyers = 5, 100
	s := startStack(t, 6, stock)
	sale, store := s.startSale(t, service.FlashConfig{Workers: 4, QueueSize: buyers})

	got := stampede(t, sale, buyers)
	if got.bought.Load() != stock || got.soldOut.Load() != buyers-stock || got.busy.Load() != 0 {
		t.Errorf("bought=%d soldOut=%d busy=%d, want %d, %d, 0", got.bought.Load(), got.soldOut.Load(), got.busy.Load(), stock, buyers-stock)
	}
	orders, left := orderCount(t, s)
	if orders != stock || left != 0 {
		t.Errorf("orders=%d database stock=%d, want %d and 0", orders, left, stock)
	}
	if calls := store.calls.Load(); calls != stock {
		t.Errorf("database order attempts = %d, want %d: the counter must stop the rest", calls, stock)
	}
	if remaining, err := s.res.Remaining(context.Background(), monitorID); err != nil || remaining != 0 {
		t.Errorf("counter = %d (err %v), want 0", remaining, err)
	}
}

// Proof 2: a small queue in front of a small pool. The shop turns people away with ErrBusy, and every turned-away
// buyer's unit goes back, so what is left on the counter is exactly stock minus orders.
func TestFlashSale_Integration_FullQueueRejectsAndKeepsTheCounterHonest(t *testing.T) {
	const stock, buyers = 300, 200
	s := startStack(t, 4, stock)
	sale, _ := s.startSale(t, service.FlashConfig{Workers: 2, QueueSize: 5})

	got := stampede(t, sale, buyers)
	if got.busy.Load() == 0 {
		t.Fatal("nobody was turned away: the queue did not push back")
	}
	if total := got.bought.Load() + got.busy.Load(); total != buyers {
		t.Errorf("bought+busy = %d, want %d", total, buyers)
	}
	orders, left := orderCount(t, s)
	if int32(orders) != got.bought.Load() || int32(stock-orders) != int32(left) {
		t.Errorf("orders=%d database stock=%d, want %d orders and %d left", orders, left, got.bought.Load(), stock-int(got.bought.Load()))
	}
	remaining, err := s.res.Remaining(context.Background(), monitorID)
	if err != nil || remaining != stock-orders {
		t.Errorf("counter = %d (err %v), want %d: turned-away buyers must give their unit back", remaining, err, stock-orders)
	}
}

// Proof 3: the pool has a ceiling and the workers stay under it. 150 buyers, 3 workers, a pool of 4: sampled while the
// stampede runs, never more than 3 connections are busy and the pool never grows past 4.
func TestFlashSale_Integration_ConnectionsStayUnderTheCeiling(t *testing.T) {
	const stock, buyers, workers, maxConns = 200, 150, 3, 4
	s := startStack(t, maxConns, stock)
	sale, _ := s.startSale(t, service.FlashConfig{Workers: workers, QueueSize: buyers})

	var peakBusy, peakTotal atomic.Int32
	stop := make(chan struct{})
	sampled := make(chan struct{})
	go func() {
		defer close(sampled)
		for {
			st := s.pool.Stat()
			if n := st.AcquiredConns(); n > peakBusy.Load() {
				peakBusy.Store(n)
			}
			if n := st.TotalConns(); n > peakTotal.Load() {
				peakTotal.Store(n)
			}
			select {
			case <-stop:
				return
			case <-time.After(time.Millisecond):
			}
		}
	}()

	got := stampede(t, sale, buyers)
	close(stop)
	<-sampled

	if got.bought.Load() != buyers {
		t.Errorf("bought = %d, want %d (stock and queue were big enough for everyone)", got.bought.Load(), buyers)
	}
	if peakBusy.Load() == 0 {
		t.Error("sampler never saw a busy connection: the measurement proves nothing")
	}
	if peakBusy.Load() > workers {
		t.Errorf("busy connections peaked at %d, want <= %d workers", peakBusy.Load(), workers)
	}
	if peakTotal.Load() > maxConns || s.pool.Stat().MaxConns() != maxConns {
		t.Errorf("pool grew to %d (max %d), want <= %d", peakTotal.Load(), s.pool.Stat().MaxConns(), maxConns)
	}
}

// The Lua script is atomic: 50 clients race for 5 units straight at the counter and exactly 5 win.
func TestReservations_Integration_ReserveIsAtomic(t *testing.T) {
	const stock, clients = 5, 50
	s := startStack(t, 2, stock)
	var won atomic.Int32
	var wg sync.WaitGroup
	for range clients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := s.res.Reserve(context.Background(), monitorID, 1)
			if err != nil {
				t.Errorf("Reserve: %v", err)
			}
			if ok {
				won.Add(1)
			}
		}()
	}
	wg.Wait()
	if won.Load() != stock {
		t.Errorf("winners = %d, want %d", won.Load(), stock)
	}
	if err := s.res.Release(context.Background(), monitorID, 2); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.res.Remaining(context.Background(), monitorID); n != 2 {
		t.Errorf("after Release(2) counter = %d, want 2", n)
	}
	if _, err := s.res.Reserve(context.Background(), 999, 1); !errors.Is(err, repository.ErrSaleNotOpen) {
		t.Errorf("unopened product: err = %v, want ErrSaleNotOpen", err)
	}
}
