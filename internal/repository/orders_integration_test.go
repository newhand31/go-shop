//go:build integration

package repository_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/newhand31/go-shop/internal/platform/postgres"
	"github.com/newhand31/go-shop/internal/repository"
	"github.com/newhand31/go-shop/internal/service"
)

const (
	monitorID = 3 // the incident product: seeded with exactly one in stock
	buyers    = 20
)

func openMigrated(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	pool, err := postgres.Open(ctx, startDB(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := postgres.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

// naiveOrders is the code that oversold the shop: read the stock, decide in Go, write the answer back.
// There is no lock, so every buyer that reads before anyone writes sees "1 left". The pause between read and write
// only makes the race certain instead of likely; the bug is in the shape of the code, not in the pause.
type naiveOrders struct{ pool *pgxpool.Pool }

func (n naiveOrders) Place(ctx context.Context, lines []repository.OrderLine) (repository.Order, error) {
	tx, err := n.pool.Begin(ctx) // note: even inside a transaction, a plain SELECT takes no lock
	if err != nil {
		return repository.Order{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, l := range lines {
		var stock int
		if err := tx.QueryRow(ctx, `SELECT stock FROM products WHERE id = $1`, l.ProductID).Scan(&stock); err != nil {
			return repository.Order{}, err
		}
		if stock < l.Quantity {
			return repository.Order{}, repository.ErrOutOfStock
		}
		time.Sleep(50 * time.Millisecond)
		// Absolute write from the value we read: the lost update. (A relative "stock = stock - n" would let the
		// CHECK constraint catch it, which is the safety net, not the fix.)
		if _, err := tx.Exec(ctx, `UPDATE products SET stock = $2 WHERE id = $1`, l.ProductID, stock-l.Quantity); err != nil {
			return repository.Order{}, err
		}
	}
	var id int
	if err := tx.QueryRow(ctx, `INSERT INTO orders (total_twd) VALUES (0) RETURNING id`).Scan(&id); err != nil {
		return repository.Order{}, err
	}
	for _, l := range lines {
		if _, err := tx.Exec(ctx, `INSERT INTO order_lines (order_id, product_id, quantity, unit_price_twd) VALUES ($1, $2, $3, 0)`, id, l.ProductID, l.Quantity); err != nil {
			return repository.Order{}, err
		}
	}
	return repository.Order{ID: id}, tx.Commit(ctx)
}

// slowBegin wraps a pool so every transaction pauses just before it writes. The real code is untouched; the pause only
// widens the gap between "check the stock" and "take it". Without FOR UPDATE every buyer would sit in that gap together.
type slowBegin struct{ pool *pgxpool.Pool }

func (s slowBegin) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := s.pool.Begin(ctx)
	return slowTx{tx}, err
}

type slowTx struct{ pgx.Tx }

func (t slowTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	time.Sleep(50 * time.Millisecond)
	return t.Tx.Exec(ctx, sql, args...)
}

// memCart is a per-buyer cart that always holds one monitor.
type memCart struct{}

func (memCart) Items(context.Context, string) (map[int]int, error) {
	return map[int]int{monitorID: 1}, nil
}
func (memCart) RemoveItem(context.Context, string, int) error { return nil }

// stampede sends `buyers` checkouts at the same instant and counts how many were told "bought".
func stampede(t *testing.T, store service.OrderStore) (bought, soldOut int) {
	t.Helper()
	svc := service.NewOrderService(store, memCart{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	var ok, no atomic.Int32
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < buyers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := svc.Checkout(context.Background(), "buyer")
			switch {
			case err == nil:
				ok.Add(1)
			case errors.Is(err, service.ErrOutOfStock):
				no.Add(1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	return int(ok.Load()), int(no.Load())
}

func countOrders(t *testing.T, pool *pgxpool.Pool) (orders, units, stock int) {
	t.Helper()
	ctx := context.Background()
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM orders`).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT coalesce(sum(quantity), 0) FROM order_lines WHERE product_id = $1`, monitorID).Scan(&units); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT stock FROM products WHERE id = $1`, monitorID).Scan(&stock); err != nil {
		t.Fatal(err)
	}
	return
}

// The incident, replayed: one monitor in stock, many buyers, no lock. More than one buyer is told "yours".
func TestOversell_WithoutALockMoreThanTheStockIsSold(t *testing.T) {
	pool := openMigrated(t)
	bought, _ := stampede(t, naiveOrders{pool})
	_, units, _ := countOrders(t, pool)
	if bought <= 1 || units <= 1 {
		t.Fatalf("bought=%d units=%d: expected the race to oversell the single monitor (that is the bug)", bought, units)
	}
	t.Logf("1 monitor in stock, %d sold", units)
}

// The fix: same stampede through repository.Orders (with a pause before each write, so removing FOR UPDATE makes this fail). Exactly one buyer wins, everyone else gets out_of_stock.
func TestOversell_WithForUpdateExactlyTheStockIsSold(t *testing.T) {
	pool := openMigrated(t)
	bought, soldOut := stampede(t, repository.NewOrders(slowBegin{pool}))
	orders, units, stock := countOrders(t, pool)
	if bought != 1 || soldOut != buyers-1 {
		t.Fatalf("bought=%d soldOut=%d, want 1 and %d", bought, soldOut, buyers-1)
	}
	if orders != 1 || units != 1 || stock != 0 {
		t.Fatalf("orders=%d units=%d stock=%d, want 1, 1, 0", orders, units, stock)
	}
}

// All or nothing: line 1 is fine, line 2 is not. The deferred Rollback must undo line 1's stock decrease.
func TestOrders_FailedOrderRollsBackEverything(t *testing.T) {
	ctx := context.Background()
	pool := openMigrated(t)
	_, err := repository.NewOrders(pool).Place(ctx, []repository.OrderLine{
		{ProductID: 1, Quantity: 2},         // 10 in stock: this line succeeds inside the transaction
		{ProductID: monitorID, Quantity: 2}, // 1 in stock: this one fails
	})
	if !errors.Is(err, repository.ErrOutOfStock) {
		t.Fatalf("err = %v, want ErrOutOfStock", err)
	}
	// No transaction may be left open holding the keyboard row: a new order for keyboards must go straight through.
	short, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if _, err := repository.NewOrders(pool).Place(short, []repository.OrderLine{{ProductID: 1, Quantity: 1}}); err != nil {
		t.Fatalf("follow-up order failed (a leaked transaction would still hold the lock): %v", err)
	}
	var keyboards, orders int
	if err := pool.QueryRow(ctx, `SELECT stock FROM products WHERE id = 1`).Scan(&keyboards); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM orders`).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	// 10 minus the one keyboard the follow-up order bought; the failed order's 2 must not be missing.
	if keyboards != 9 || orders != 1 {
		t.Fatalf("keyboard stock=%d orders=%d, want 9 and 1: the first line leaked out of the rolled-back transaction", keyboards, orders)
	}
}

// Two buyers ask for the same two products in opposite order. Locks are taken sorted by id, so neither can deadlock.
func TestOrders_OppositeLineOrderDoesNotDeadlock(t *testing.T) {
	pool := openMigrated(t)
	orders := repository.NewOrders(pool)
	var wg sync.WaitGroup
	for _, lines := range [][]repository.OrderLine{
		{{ProductID: 1, Quantity: 1}, {ProductID: 2, Quantity: 1}},
		{{ProductID: 2, Quantity: 1}, {ProductID: 1, Quantity: 1}},
	} {
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := orders.Place(context.Background(), lines); err != nil {
					t.Errorf("Place: %v", err)
				}
			}()
		}
	}
	wg.Wait()
}

func TestOrders_UnknownProduct(t *testing.T) {
	pool := openMigrated(t)
	_, err := repository.NewOrders(pool).Place(context.Background(), []repository.OrderLine{{ProductID: 999, Quantity: 1}})
	if !errors.Is(err, repository.ErrProductNotFound) {
		t.Fatalf("err = %v, want ErrProductNotFound", err)
	}
}
