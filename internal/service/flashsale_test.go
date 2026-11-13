package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/newhand31/go-shop/internal/repository"
)

// flashGate is a counter in memory with the same contract as the Redis one. A mutex stands in for Lua's atomicity.
type flashGate struct {
	mu       sync.Mutex
	left     int
	released int
}

func (g *flashGate) Reserve(_ context.Context, _, qty int) (bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.left < qty {
		return false, nil
	}
	g.left -= qty
	return true, nil
}

func (g *flashGate) Release(_ context.Context, _, qty int) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.left += qty
	g.released += qty
	return nil
}

func (g *flashGate) state() (left, released int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.left, g.released
}

// flashStore is the database. It counts how many calls are inside Place at once and how many calls there were.
type flashStore struct {
	hold    func(ctx context.Context) // what Place does while "in the database"
	err     error
	calls   atomic.Int32
	current atomic.Int32
	peak    atomic.Int32
}

func (s *flashStore) Place(ctx context.Context, lines []repository.OrderLine) (repository.Order, error) {
	s.calls.Add(1)
	n := s.current.Add(1)
	defer s.current.Add(-1)
	for {
		p := s.peak.Load()
		if n <= p || s.peak.CompareAndSwap(p, n) {
			break
		}
	}
	if s.hold != nil {
		s.hold(ctx)
	}
	if s.err != nil {
		return repository.Order{}, s.err
	}
	return repository.Order{ID: 1, TotalTWD: 100, Lines: []repository.OrderedLine{{ProductID: lines[0].ProductID, Quantity: lines[0].Quantity, UnitPriceTWD: 100}}}, nil
}

func newSale(t *testing.T, gate StockGate, store OrderStore, cfg FlashConfig) *FlashSale {
	t.Helper()
	f := NewFlashSale(gate, store, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); f.Run(ctx) }()
	t.Cleanup(func() { cancel(); <-done })
	return f
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

// buyAsync starts Buy in its own goroutine and hands back the channel for its error.
func buyAsync(ctx context.Context, f *FlashSale) <-chan error {
	out := make(chan error, 1)
	go func() {
		_, err := f.Buy(ctx, 3, 1)
		out <- err
	}()
	return out
}

// The queue is full: one buyer inside the database, one waiting, the third is turned away at once, and the database
// never hears about it.
func TestFlashSale_FullQueueRejectsWithoutTouchingTheDatabase(t *testing.T) {
	gate := &flashGate{left: 10}
	release := make(chan struct{})
	store := &flashStore{hold: func(context.Context) { <-release }}
	f := newSale(t, gate, store, FlashConfig{Workers: 1, QueueSize: 1})

	first := buyAsync(context.Background(), f)
	waitFor(t, "first buyer to reach the database", func() bool { return store.current.Load() == 1 })
	second := buyAsync(context.Background(), f)
	waitFor(t, "second buyer to take the queue slot", func() bool { return len(f.queue) == 1 })

	// The queue is full. This call must come back now, not when the database is free.
	start := time.Now()
	_, err := f.Buy(context.Background(), 3, 1)
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("third buyer: err = %v, want ErrBusy", err)
	}
	if time.Since(start) > time.Second {
		t.Errorf("rejection took %v: it waited instead of answering", time.Since(start))
	}
	if left, released := gate.state(); left != 8 || released != 1 {
		t.Errorf("gate left=%d released=%d, want 8 and 1: the rejected buyer must give its unit back", left, released)
	}
	if got := store.calls.Load(); got != 1 {
		t.Errorf("database calls = %d, want 1: the rejected buyer must not reach it", got)
	}

	close(release)
	for i, ch := range []<-chan error{first, second} {
		if err := <-ch; err != nil {
			t.Errorf("buyer %d: %v", i+1, err)
		}
	}
}

// However many buyers arrive, no more than Workers are inside the database at the same moment.
func TestFlashSale_DatabaseConcurrencyNeverExceedsWorkers(t *testing.T) {
	const workers, buyers = 3, 40
	gate := &flashGate{left: buyers}
	store := &flashStore{hold: func(context.Context) { time.Sleep(5 * time.Millisecond) }}
	f := newSale(t, gate, store, FlashConfig{Workers: workers, QueueSize: buyers})

	var wg sync.WaitGroup
	for range buyers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := f.Buy(context.Background(), 3, 1); err != nil {
				t.Errorf("Buy: %v", err)
			}
		}()
	}
	wg.Wait()
	if peak := store.peak.Load(); peak > workers {
		t.Errorf("peak in database = %d, want <= %d", peak, workers)
	}
	if got := store.calls.Load(); got != buyers {
		t.Errorf("database calls = %d, want %d", got, buyers)
	}
}

// The counter is the gate: with 5 units and 50 buyers, 45 are told "sold out" and only 5 ever reach the database.
func TestFlashSale_SoldOutAtTheCounterNeverReachesTheDatabase(t *testing.T) {
	const stock, buyers = 5, 50
	gate := &flashGate{left: stock}
	store := &flashStore{}
	f := newSale(t, gate, store, FlashConfig{Workers: 2, QueueSize: buyers})

	var bought, soldOut atomic.Int32
	var wg sync.WaitGroup
	for range buyers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := f.Buy(context.Background(), 3, 1)
			switch {
			case err == nil:
				bought.Add(1)
			case errors.Is(err, ErrOutOfStock):
				soldOut.Add(1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
	if bought.Load() != stock || soldOut.Load() != buyers-stock {
		t.Errorf("bought=%d soldOut=%d, want %d and %d", bought.Load(), soldOut.Load(), stock, buyers-stock)
	}
	if got := store.calls.Load(); got != stock {
		t.Errorf("database calls = %d, want %d", got, stock)
	}
}

// The compensation: when the database says no, the unit goes back to the counter so the next buyer can have it.
func TestFlashSale_DatabaseFailureGivesTheUnitBack(t *testing.T) {
	gate := &flashGate{left: 1}
	store := &flashStore{err: errors.New("connection reset")}
	f := newSale(t, gate, store, FlashConfig{Workers: 1, QueueSize: 1})

	if _, err := f.Buy(context.Background(), 3, 1); err == nil {
		t.Fatal("Buy succeeded, want the database error")
	}
	if left, released := gate.state(); left != 1 || released != 1 {
		t.Errorf("gate left=%d released=%d, want 1 and 1", left, released)
	}
}

// A buyer who hangs up while queued gives the unit back, and the worker skips the corpse instead of placing an order
// nobody is waiting for.
func TestFlashSale_BuyerLeavesWhileQueued_UnitReturnedAndOrderSkipped(t *testing.T) {
	gate := &flashGate{left: 10}
	release := make(chan struct{})
	store := &flashStore{hold: func(context.Context) { <-release }}
	f := newSale(t, gate, store, FlashConfig{Workers: 1, QueueSize: 2})

	first := buyAsync(context.Background(), f)
	waitFor(t, "first buyer to reach the database", func() bool { return store.current.Load() == 1 })
	ctx, hangUp := context.WithCancel(context.Background())
	second := buyAsync(ctx, f)
	waitFor(t, "second buyer to queue", func() bool { return len(f.queue) == 1 })

	hangUp()
	if err := <-second; !errors.Is(err, context.Canceled) {
		t.Fatalf("second buyer: err = %v, want context.Canceled", err)
	}
	if left, released := gate.state(); left != 9 || released != 1 {
		t.Errorf("gate left=%d released=%d, want 9 and 1", left, released)
	}

	close(release)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	waitFor(t, "worker to pop the corpse", func() bool { return len(f.queue) == 0 })
	if got := store.calls.Load(); got != 1 {
		t.Errorf("database calls = %d, want 1: the abandoned order must be skipped", got)
	}
}

func TestFlashSale_RejectsNonPositiveQuantity(t *testing.T) {
	f := NewFlashSale(&flashGate{left: 1}, &flashStore{}, FlashConfig{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	var v *ValidationError
	if _, err := f.Buy(context.Background(), 3, 0); !errors.As(err, &v) {
		t.Errorf("err = %v, want *ValidationError", err)
	}
}

// Shutdown: once Run has returned, a queued buyer is not stuck forever and gets its unit back.
func TestFlashSale_StoppedWhileQueued_BuyerIsReleased(t *testing.T) {
	gate := &flashGate{left: 1}
	f := NewFlashSale(gate, &flashStore{}, FlashConfig{Workers: 1, QueueSize: 1}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); f.Run(ctx) }()
	cancel()
	<-done

	if _, err := f.Buy(context.Background(), 3, 1); !errors.Is(err, ErrBusy) {
		t.Fatalf("err = %v, want ErrBusy", err)
	}
	if left, _ := gate.state(); left != 1 {
		t.Errorf("gate left = %d, want 1", left)
	}
}
