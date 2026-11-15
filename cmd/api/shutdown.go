package main

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/newhand31/go-shop/internal/platform/lifecycle"
)

// defaultShutdownGrace is how long running requests get to finish after a stop signal. It must be shorter than the
// platform's own patience (docker stop waits 10 s by default; docker-compose.yml raises it to 40 s: 25 s drain plus the 5 s span flush), or the platform
// sends SIGKILL in the middle of our graceful shutdown.
const defaultShutdownGrace = 25 * time.Second

// serveThenStop is the whole shutdown order in one place, and main calls exactly this:
//
//  1. ctx is cancelled (SIGINT/SIGTERM): the listener closes, running requests are allowed to finish (up to grace);
//  2. only then the background goroutines are stopped and awaited. They come second because a running request may still
//     need them: a flash-sale buyer waits for a worker. Stop them first and that buyer is turned away mid-purchase.
//
// Afterwards run returns and its defers close Redis, the pool and flush the spans, in that order.
func serveThenStop(ctx context.Context, srv *http.Server, ln net.Listener, grace time.Duration, workers *lifecycle.Group) error {
	err := lifecycle.Serve(ctx, srv, ln, grace)
	workers.Stop()
	return err
}
