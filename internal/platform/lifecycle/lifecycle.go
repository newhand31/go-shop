// Package lifecycle holds the two halves of a clean shutdown: stop taking requests but let the running ones finish
// (Serve), then stop the background goroutines and wait until each one has returned (Group).
package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// Serve runs srv on ln until it fails or ctx is cancelled (a signal, in main). On cancel it calls srv.Shutdown: the
// listener closes at once, so no new connection gets in; requests already running keep going and Serve returns when
// the last one has answered. If they are not done after grace, the remaining connections are cut and the error says so.
//
// The trap this function exists for: the moment Shutdown starts, srv.Serve returns http.ErrServerClosed. That is not
// "finished", it is "started closing". Code that returns on that error exits while the drain is still going.
func Serve(ctx context.Context, srv *http.Server, ln net.Listener, grace time.Duration) error {
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	// ctx is already cancelled, so the drain needs its own clock, or Shutdown would give up on its first line.
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), grace)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = srv.Close() // out of time: cut what is left
		<-serveErr
		return fmt.Errorf("graceful shutdown (grace %s): %w", grace, err)
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

// Group owns background goroutines. Its context is NOT derived from the signal context: a goroutine that in-flight
// requests still depend on (the flash-sale workers) must outlive the signal, and only stop when Stop is called after
// the server has drained.
type Group struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewGroup returns a group whose goroutines run until Stop.
func NewGroup() *Group {
	ctx, cancel := context.WithCancel(context.Background())
	return &Group{ctx: ctx, cancel: cancel}
}

// Go starts fn in its own goroutine with the group's context. Do not call Go after Stop.
func (g *Group) Go(fn func(ctx context.Context)) {
	g.wg.Go(func() { fn(g.ctx) })
}

// Stop cancels the context and waits until every goroutine has returned. It is safe to call more than once, so main can
// both call it in the normal path and defer it for the early returns. It has no timeout: a goroutine that ignores its
// context makes Stop wait, and the platform's hard kill ends the wait.
func (g *Group) Stop() {
	g.cancel()
	g.wg.Wait()
}
