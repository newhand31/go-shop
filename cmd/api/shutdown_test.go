package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/newhand31/go-shop/internal/handler"
	"github.com/newhand31/go-shop/internal/platform/lifecycle"
	"github.com/newhand31/go-shop/internal/repository"
	"github.com/newhand31/go-shop/internal/service"
)

// countingGate is a flash-sale counter in memory that never runs out.
type countingGate struct{ reserves atomic.Int32 }

func (g *countingGate) Reserve(context.Context, int, int) (bool, error) {
	g.reserves.Add(1)
	return true, nil
}
func (g *countingGate) Release(context.Context, int, int) error { return nil }

// gatedStore is a database whose first order stays "in the transaction" until release is closed.
type gatedStore struct {
	calls   atomic.Int32
	entered chan struct{}
	release chan struct{}
}

func (s *gatedStore) Place(_ context.Context, lines []repository.OrderLine) (repository.Order, error) {
	if s.calls.Add(1) == 1 {
		close(s.entered)
		<-s.release
	}
	return repository.Order{ID: int(s.calls.Load()), TotalTWD: 100, Lines: []repository.OrderedLine{{ProductID: lines[0].ProductID, Quantity: 1, UnitPriceTWD: 100}}}, nil
}

// TestStopSignalLetsQueuedBuyersFinish uses the same two calls as main (Group for the workers, serveThenStop for the
// order). One worker is busy with buyer A and buyer B waits behind it in the queue when the stop signal arrives. If the
// workers had been told to stop together with the server, B would be turned away with a 503 even though its request was
// accepted before the signal. Both must get their order.
func TestStopSignalLetsQueuedBuyersFinish(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gate := &countingGate{}
	store := &gatedStore{entered: make(chan struct{}), release: make(chan struct{})}
	sale := service.NewFlashSale(gate, store, service.FlashConfig{Workers: 1, QueueSize: 4}, logger)

	workers := lifecycle.NewGroup()
	defer workers.Stop()
	workers.Go(sale.Run)

	srv := &http.Server{Handler: handler.WithFlashSale(http.NotFoundHandler(), sale, logger)}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	signal, stop := context.WithCancel(context.Background())
	defer stop()
	served := make(chan error, 1)
	go func() { served <- serveThenStop(signal, srv, ln, 5*time.Second, workers) }()

	statuses := make(chan int, 2)
	buy := func() {
		client := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
		resp, err := client.Post("http://"+ln.Addr().String()+"/flash-sales/1/orders", "application/json", strings.NewReader("{}"))
		if err != nil {
			t.Errorf("buy: %v", err)
			statuses <- 0
			return
		}
		_ = resp.Body.Close()
		statuses <- resp.StatusCode
	}

	go buy() // A: taken by the only worker, stuck in the transaction
	waitFor(t, store.entered, "buyer A to reach the database")
	go buy() // B: accepted, waiting in the queue behind A
	deadline := time.Now().Add(5 * time.Second)
	for gate.reserves.Load() < 2 {
		if time.Now().After(deadline) {
			t.Fatal("buyer B never reached the shop")
		}
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond) // B has passed the counter; give it a moment to land in the queue

	stop() // the stop signal arrives with A in the database and B in the queue
	time.Sleep(100 * time.Millisecond)
	close(store.release)

	for range 2 {
		select {
		case code := <-statuses:
			if code != http.StatusCreated {
				t.Errorf("status = %d, want 201 for a request accepted before the stop signal", code)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("a buyer never got an answer")
		}
	}
	select {
	case err := <-served:
		if err != nil {
			t.Fatalf("serveThenStop = %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serveThenStop did not return after the drain")
	}
}

// TestWorkersAreStoppedBeforeServeThenStopReturns: when it returns, run's defers may close the pool, so no worker may
// still be running.
func TestWorkersAreStoppedBeforeServeThenStopReturns(t *testing.T) {
	workers := lifecycle.NewGroup()
	var running sync.WaitGroup
	var stillRunning atomic.Bool
	running.Add(1)
	workers.Go(func(ctx context.Context) {
		defer running.Done()
		<-ctx.Done()
		time.Sleep(50 * time.Millisecond) // finishing its last round
		stillRunning.Store(false)
	})
	stillRunning.Store(true)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithCancel(context.Background())
	stop()
	if err := serveThenStop(ctx, &http.Server{}, ln, time.Second, workers); err != nil {
		t.Fatal(err)
	}
	if stillRunning.Load() {
		t.Fatal("serveThenStop returned while a worker was still running")
	}
	running.Wait()
}

// TestDockerfileKeepsTheShutdownPromises reads the Dockerfile as text. It does not build an image; it only guards the
// three lines that decide whether graceful shutdown works inside a container.
func TestDockerfileKeepsTheShutdownPromises(t *testing.T) {
	raw, err := os.ReadFile("../../Dockerfile")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	var from, entry string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(strings.ToUpper(l), "FROM "):
			from = l // the last FROM is the image that ships
		case strings.HasPrefix(strings.ToUpper(l), "ENTRYPOINT"):
			entry = l
		}
	}
	if !strings.HasPrefix(from, "FROM scratch") {
		t.Errorf("the final stage is %q, want FROM scratch", from)
	}
	// Exec form puts our binary at PID 1, so docker stop's SIGTERM reaches it. Shell form would put /bin/sh there.
	if !strings.HasPrefix(entry, "ENTRYPOINT [") {
		t.Errorf("ENTRYPOINT = %q, want the exec (JSON array) form", entry)
	}
	if !strings.Contains(string(raw), "CGO_ENABLED=0") {
		t.Error("the build must set CGO_ENABLED=0: scratch has no libc to link against")
	}
}

func waitFor(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}
