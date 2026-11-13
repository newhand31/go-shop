// Command api is the HTTP API entry point of go-shop.
//
// All dependencies are wired by hand in main: no DI container, the whole wiring fits on one screen.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/newhand31/go-shop/internal/handler"
	"github.com/newhand31/go-shop/internal/platform/postgres"
	"github.com/newhand31/go-shop/internal/platform/ratelimit"
	"github.com/newhand31/go-shop/internal/platform/redis"
	"github.com/newhand31/go-shop/internal/platform/session"
	"github.com/newhand31/go-shop/internal/repository"
	"github.com/newhand31/go-shop/internal/service"
)

// Per client IP: 10 requests a second on average, bursts up to 20. A shop page makes a handful of calls, a script makes thousands.
const (
	rateLimitPerSecond = 10
	rateLimitBurst     = 20
	sessionTTL         = 24 * time.Hour
)

// Implicit interfaces, checked where the pieces meet. Nobody wrote "implements": these lines only ask the compiler
// whether the methods fit. Delete a method from session.Redis or repository.Carts and the build fails here.
var (
	_ handler.SessionStore = (*session.Redis)(nil)
	_ service.CartStore    = (*repository.Carts)(nil)
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("api stopped", "err", err)
		os.Exit(1)
	}
}

// run returns errors instead of calling os.Exit, so every defer (closing the pool) actually runs.
func run(logger *slog.Logger) error {
	// Ctrl+C cancels ctx. Draining in-flight requests properly is episode 12; here we only make sure the pool closes.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL is not set")
	}
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return errors.New("REDIS_URL is not set")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	pool, err := postgres.Open(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := postgres.Migrate(ctx, pool); err != nil {
		return err
	}

	rdb, err := redis.Open(ctx, redisURL)
	if err != nil {
		return err
	}
	defer rdb.Close()

	// Build from the inside out: repository → service → handler.
	repo := repository.NewProducts(pool)
	svc := service.NewProductService(repo)
	// Middleware order matters: the rate limiter sits before the session store, so a flood never gets to create sessions.
	limiter := ratelimit.New(rateLimitPerSecond, rateLimitBurst)
	sessions := session.NewRedis(rdb, sessionTTL) // was session.NewMemory in episode 4: the handler cannot tell
	carts := service.NewCartService(repository.NewCarts(rdb), svc)
	h := handler.New(svc, carts, logger,
		handler.RateLimit(limiter, logger),
		handler.Session(sessions, sessionTTL, logger),
	)

	srv := &http.Server{Addr: ":" + port, Handler: h}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	logger.Info("api listening", "port", port)

	select {
	case err := <-errCh:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
		logger.Info("interrupt received, closing")
		return srv.Close()
	}
}
