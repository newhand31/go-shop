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
	"strconv"
	"time"

	"github.com/newhand31/go-shop/internal/handler"
	"github.com/newhand31/go-shop/internal/payment/ecpay"
	"github.com/newhand31/go-shop/internal/platform/postgres"
	"github.com/newhand31/go-shop/internal/platform/ratelimit"
	"github.com/newhand31/go-shop/internal/platform/redis"
	"github.com/newhand31/go-shop/internal/platform/session"
	"github.com/newhand31/go-shop/internal/repository"
	"github.com/newhand31/go-shop/internal/service"
)

// Per client IP: 10 requests a second on average, bursts up to 20. A shop page makes a handful of calls, a script makes thousands.
// A load test comes from one IP, so RATE_LIMIT_RPS and RATE_LIMIT_BURST can raise these for that run (never in production).
const (
	defaultRateLimitPerSecond = 10
	defaultRateLimitBurst     = 20
	sessionTTL                = 24 * time.Hour
)

// envInt reads a positive integer from the environment, or returns def when it is unset or invalid.
func envInt(name string, def int) int {
	if n, err := strconv.Atoi(os.Getenv(name)); err == nil && n > 0 {
		return n
	}
	return def
}

// Implicit interfaces, checked where the pieces meet. Nobody wrote "implements": these lines only ask the compiler
// whether the methods fit. Delete a method from session.Redis or repository.Carts and the build fails here.
var (
	_ handler.SessionStore     = (*session.Redis)(nil)
	_ service.CartStore        = (*repository.Carts)(nil)
	_ service.CheckoutCart     = (*repository.Carts)(nil)
	_ service.OrderStore       = (*repository.Orders)(nil)
	_ service.PaymentStore     = (*repository.Payments)(nil)
	_ service.CallbackVerifier = (*ecpay.Verifier)(nil)
	_ service.OutboxStore      = (*repository.Outbox)(nil)
	_ service.Mailer           = logMailer{}
)

// logMailer stands in for a real email provider: the shop has no customer email address yet (there are no accounts),
// so it only writes what it would send. It is a Mailer because it has the method, nothing else.
type logMailer struct{ logger *slog.Logger }

func (m logMailer) SendOrderConfirmation(_ context.Context, orderID, totalTWD int) error {
	m.logger.Info("would send order confirmation", "order_id", orderID, "total_twd", totalTWD)
	return nil
}

// ecpayVerifier reads the merchant's secrets from the environment and fails closed: with no secrets the API refuses to
// start, because a callback endpoint verified with a published key accepts anyone's forged "paid". The demo opts in
// to ECPay's public sandbox credentials explicitly with ECPAY_SANDBOX=1.
func ecpayVerifier(logger *slog.Logger) (*ecpay.Verifier, error) {
	id, key, iv := os.Getenv("ECPAY_MERCHANT_ID"), os.Getenv("ECPAY_HASH_KEY"), os.Getenv("ECPAY_HASH_IV")
	if id != "" && key != "" && iv != "" {
		return ecpay.NewVerifier(id, key, iv), nil
	}
	if os.Getenv("ECPAY_SANDBOX") == "1" {
		logger.Warn("ECPAY_SANDBOX=1: using ECPay's public sandbox credentials, test payments only")
		return ecpay.NewVerifier(ecpay.SandboxMerchantID, ecpay.SandboxHashKey, ecpay.SandboxHashIV), nil
	}
	return nil, errors.New("set ECPAY_MERCHANT_ID, ECPAY_HASH_KEY and ECPAY_HASH_IV (or ECPAY_SANDBOX=1 for the public test credentials)")
}

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
	limiter := ratelimit.New(float64(envInt("RATE_LIMIT_RPS", defaultRateLimitPerSecond)), envInt("RATE_LIMIT_BURST", defaultRateLimitBurst))
	sessions := session.NewRedis(rdb, sessionTTL) // was session.NewMemory in episode 4: the handler cannot tell
	carts := service.NewCartService(repository.NewCarts(rdb), svc)
	orders := service.NewOrderService(repository.NewOrders(pool), repository.NewCarts(rdb), logger)
	h := handler.New(svc, carts, orders, logger,
		handler.RateLimit(limiter, logger),
		handler.Session(sessions, sessionTTL, logger),
	)

	verifier, err := ecpayVerifier(logger)
	if err != nil {
		return err
	}
	payments := service.NewPaymentService(verifier, repository.NewPayments(pool), logger)

	// The outbox worker is a goroutine we start, so we must stop it and wait for it. The deferred function runs before
	// the deferred pool.Close above: cancel, then wait until Run has returned, so it never touches a closed pool.
	dispatcher := service.NewOutboxDispatcher(repository.NewOutbox(pool), logMailer{logger}, service.OutboxConfig{}, logger)
	workerCtx, stopWorker := context.WithCancel(ctx)
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		dispatcher.Run(workerCtx)
	}()
	defer func() {
		stopWorker()
		<-workerDone
	}()

	srv := &http.Server{Addr: ":" + port, Handler: handler.WithPaymentCallback(h, payments, logger)}
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
