package lifecycle_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/newhand31/go-shop/internal/platform/lifecycle"
)

const wait = 5 * time.Second // generous: only a hang should ever hit it

func listen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

type getResult struct {
	status int
	body   string
	err    error
}

// get sends one request on its own goroutine and reports the outcome on a channel.
func get(url string) <-chan getResult {
	out := make(chan getResult, 1)
	go func() {
		client := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
		resp, err := client.Get(url)
		if err != nil {
			out <- getResult{err: err}
			return
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		out <- getResult{status: resp.StatusCode, body: string(b), err: err}
	}()
	return out
}

func recv[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(wait):
		t.Fatalf("timed out waiting for %s", what)
		panic("unreachable")
	}
}

// blockingServer answers "done" only after release is closed. entered says a request is inside the handler.
func blockingServer() (srv *http.Server, entered chan struct{}, release chan struct{}) {
	entered = make(chan struct{}, 8)
	release = make(chan struct{})
	srv = &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		<-release
		_, _ = io.WriteString(w, "done")
	})}
	return srv, entered, release
}

func TestServeLetsAnInFlightRequestFinishBeforeReturning(t *testing.T) {
	srv, entered, release := blockingServer()
	ln := listen(t)
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	served := make(chan error, 1)
	go func() { served <- lifecycle.Serve(ctx, srv, ln, wait) }()

	resp := get("http://" + ln.Addr().String())
	recv(t, entered, "the request to reach the handler")

	stop() // the signal arrives while the request is in the handler

	select {
	case err := <-served:
		t.Fatalf("Serve returned (%v) while a request was still running", err)
	case <-time.After(200 * time.Millisecond):
	}

	close(release)
	if got := recv(t, resp, "the response"); got.err != nil || got.status != http.StatusOK || got.body != "done" {
		t.Fatalf("response = %+v, want 200 done", got)
	}
	if err := recv(t, served, "Serve to return"); err != nil {
		t.Fatalf("Serve = %v, want nil after a clean drain", err)
	}
}

func TestServeRefusesNewConnectionsOnceShutdownStarts(t *testing.T) {
	srv, entered, release := blockingServer()
	defer close(release)
	ln := listen(t)
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	served := make(chan error, 1)
	go func() { served <- lifecycle.Serve(ctx, srv, ln, wait) }()

	get("http://" + ln.Addr().String())
	recv(t, entered, "the first request to reach the handler")
	stop()

	deadline := time.Now().Add(wait)
	for {
		conn, err := net.DialTimeout("tcp", ln.Addr().String(), 100*time.Millisecond)
		if err != nil {
			return // the listener is closed: no new customers, while the old one is still being served
		}
		_ = conn.Close()
		if time.Now().After(deadline) {
			t.Fatal("the listener still accepts connections after shutdown started")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestServeCutsConnectionsWhenGraceRunsOut(t *testing.T) {
	srv, entered, release := blockingServer()
	defer close(release) // the handler never finishes on its own within the grace period
	ln := listen(t)
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	served := make(chan error, 1)
	go func() { served <- lifecycle.Serve(ctx, srv, ln, 100*time.Millisecond) }()

	resp := get("http://" + ln.Addr().String())
	recv(t, entered, "the request to reach the handler")
	stop()

	err := recv(t, served, "Serve to give up")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Serve = %v, want an error wrapping context.DeadlineExceeded", err)
	}
	if got := recv(t, resp, "the client to notice"); got.err == nil {
		t.Fatalf("client got %+v, want a broken connection", got)
	}
}

func TestServeReturnsTheErrorWhenTheServerFailsOnItsOwn(t *testing.T) {
	ln := listen(t)
	_ = ln.Close() // Serve on a closed listener fails at once
	err := lifecycle.Serve(context.Background(), &http.Server{}, ln, time.Second)
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("Serve = %v, want a real error", err)
	}
}

func TestServeWithNothingInFlightReturnsNilAtOnce(t *testing.T) {
	ln := listen(t)
	ctx, stop := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- lifecycle.Serve(ctx, &http.Server{}, ln, wait) }()
	stop()
	if err := recv(t, served, "Serve to return"); err != nil {
		t.Fatalf("Serve = %v, want nil", err)
	}
}

func TestGroupStopWaitsForEveryGoroutine(t *testing.T) {
	g := lifecycle.NewGroup()
	var finished atomic.Int32
	for range 3 {
		g.Go(func(ctx context.Context) {
			<-ctx.Done()
			time.Sleep(50 * time.Millisecond) // cleanup that takes a moment
			finished.Add(1)
		})
	}
	g.Stop()
	if n := finished.Load(); n != 3 {
		t.Fatalf("%d of 3 goroutines had finished when Stop returned", n)
	}
	g.Stop() // a second Stop (the deferred one in main) must be harmless
}
