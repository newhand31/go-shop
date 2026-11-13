//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/newhand31/go-shop/internal/repository"
	"github.com/newhand31/go-shop/internal/service"
)

func stockOf(t *testing.T, pool *pgxpool.Pool, id int) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT stock FROM products WHERE id = $1`, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// laterJanitor sees the world one hour from now, so every order we just placed is "old" without waiting or backdating.
func laterJanitor(pool *pgxpool.Pool) *service.ExpiryJanitor {
	return service.NewExpiryJanitor(repository.NewExpiry(pool), service.ExpiryConfig{
		OrderTTL: 15 * time.Minute,
		Now:      func() time.Time { return time.Now().Add(time.Hour) },
	}, quietLogger())
}

func TestExpiry_UnpaidOrderIsReleasedAndStockComesBack(t *testing.T) {
	ctx := context.Background()
	pool := openMigrated(t)
	order, err := repository.NewOrders(pool).Place(ctx, []repository.OrderLine{{ProductID: 1, Quantity: 2}, {ProductID: 3, Quantity: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if stockOf(t, pool, 1) != 8 || stockOf(t, pool, 3) != 0 {
		t.Fatal("setup: the order should have taken 2 of product 1 and the last monitor")
	}

	if got := laterJanitor(pool).Sweep(ctx); got != 1 {
		t.Fatalf("released %d orders, want 1", got)
	}
	if got := stockOf(t, pool, 1); got != 10 {
		t.Errorf("product 1 stock = %d, want 10", got)
	}
	if got := stockOf(t, pool, 3); got != 1 {
		t.Errorf("monitor stock = %d, want the last one back (1)", got)
	}
	var expired bool
	if err := pool.QueryRow(ctx, `SELECT expired_at IS NOT NULL FROM orders WHERE id = $1`, order.ID).Scan(&expired); err != nil || !expired {
		t.Fatalf("expired_at not set (err %v)", err)
	}

	// Releasing is once only: a second sweep must not hand the stock back again.
	if got := laterJanitor(pool).Sweep(ctx); got != 0 {
		t.Errorf("second sweep released %d, want 0", got)
	}
	if got := stockOf(t, pool, 1); got != 10 {
		t.Errorf("product 1 stock = %d after a second sweep, want still 10", got)
	}
}

func TestExpiry_PaidOrderIsLeftAlone(t *testing.T) {
	ctx := context.Background()
	pool := openMigrated(t)
	order, err := repository.NewOrders(pool).Place(ctx, []repository.OrderLine{{ProductID: 1, Quantity: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.NewPayments(pool).Record(ctx, repository.Payment{OrderID: order.ID, TradeNo: "T-PAID", AmountTWD: order.TotalTWD}); err != nil {
		t.Fatal(err)
	}

	if got := laterJanitor(pool).Sweep(ctx); got != 0 {
		t.Fatalf("released %d orders, want 0: the only order is paid", got)
	}
	if got := stockOf(t, pool, 1); got != 8 {
		t.Errorf("product 1 stock = %d, want 8: a paid order keeps its stock", got)
	}
}

func TestExpiry_RecentUnpaidOrderIsLeftAlone(t *testing.T) {
	ctx := context.Background()
	pool := openMigrated(t)
	if _, err := repository.NewOrders(pool).Place(ctx, []repository.OrderLine{{ProductID: 1, Quantity: 2}}); err != nil {
		t.Fatal(err)
	}
	real := service.NewExpiryJanitor(repository.NewExpiry(pool), service.ExpiryConfig{OrderTTL: 15 * time.Minute}, quietLogger())
	if got := real.Sweep(ctx); got != 0 {
		t.Fatalf("released %d orders, want 0: the order is seconds old", got)
	}
	if got := stockOf(t, pool, 1); got != 8 {
		t.Errorf("stock = %d, want 8", got)
	}
}
