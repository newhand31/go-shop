package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// fakeExpiryStore releases `orders` orders, then reports none left. block makes every call wait for its context, the way
// a query stuck on a lock would.
type fakeExpiryStore struct {
	orders  int
	block   bool
	calls   atomic.Int32
	cutoffs []time.Time
	ctxErr  error
	hadDL   bool
}

func (f *fakeExpiryStore) ReleaseOne(ctx context.Context, cutoff time.Time) (int, bool, error) {
	f.calls.Add(1)
	f.cutoffs = append(f.cutoffs, cutoff)
	_, f.hadDL = ctx.Deadline()
	if f.block {
		<-ctx.Done()
		f.ctxErr = ctx.Err()
		return 0, false, ctx.Err()
	}
	if f.orders == 0 {
		return 0, false, nil
	}
	f.orders--
	return f.orders + 1, true, nil
}

func TestExpiryJanitor_SweepReleasesUntilNoneLeftAndUsesTheClock(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	store := &fakeExpiryStore{orders: 3}
	j := NewExpiryJanitor(store, ExpiryConfig{OrderTTL: 15 * time.Minute, Now: func() time.Time { return now }}, quiet())

	if got := j.Sweep(context.Background()); got != 3 {
		t.Fatalf("released %d, want 3", got)
	}
	if want := now.Add(-15 * time.Minute); !store.cutoffs[0].Equal(want) {
		t.Fatalf("cutoff = %v, want %v (injected now minus the TTL)", store.cutoffs[0], want)
	}
	if !store.hadDL {
		t.Fatal("the store's context carries no deadline: the round is unbounded")
	}
}

// The timeout test: a round that gets stuck must be cancelled by its own deadline, and the janitor must come back
// instead of hanging with it.
func TestExpiryJanitor_SweepIsCancelledWhenTheRoundTimesOut(t *testing.T) {
	store := &fakeExpiryStore{block: true}
	j := NewExpiryJanitor(store, ExpiryConfig{RoundTimeout: 20 * time.Millisecond}, quiet())

	done := make(chan int, 1)
	go func() { done <- j.Sweep(context.Background()) }()
	select {
	case got := <-done:
		if got != 0 {
			t.Fatalf("released %d, want 0", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Sweep did not return: the round timeout did not cancel it")
	}
	if !errors.Is(store.ctxErr, context.DeadlineExceeded) {
		t.Fatalf("store saw %v, want DeadlineExceeded", store.ctxErr)
	}
}

func TestExpiryJanitor_RunStopsWhenContextIsCancelled(t *testing.T) {
	store := &fakeExpiryStore{}
	j := NewExpiryJanitor(store, ExpiryConfig{Interval: time.Millisecond}, quiet())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		j.Run(ctx)
	}()
	for wait := time.Now(); store.calls.Load() < 3; time.Sleep(time.Millisecond) {
		if time.Since(wait) > 2*time.Second {
			t.Fatal("Run never looped")
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}
