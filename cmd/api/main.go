// Command api is the HTTP API entry point of go-shop.
//
// All dependencies are wired by hand in main: no DI container, the whole wiring fits on one screen.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"

	"github.com/newhand31/go-shop/internal/handler"
	"github.com/newhand31/go-shop/internal/payment/ecpay"
	"github.com/newhand31/go-shop/internal/platform/lifecycle"
	"github.com/newhand31/go-shop/internal/platform/postgres"
	"github.com/newhand31/go-shop/internal/platform/ratelimit"
	"github.com/newhand31/go-shop/internal/platform/redis"
	"github.com/newhand31/go-shop/internal/platform/session"
	"github.com/newhand31/go-shop/internal/platform/telemetry"
	"github.com/newhand31/go-shop/internal/repository"
	"github.com/newhand31/go-shop/internal/service"
)

// Per client IP: 10 requests a second on average, bursts up to 20. A shop page makes a handful of calls, a script makes thousands.
// A load test comes from one IP, so RATE_LIMIT_RPS and RATE_LIMIT_BURST can raise these for that run (never in production).
const (
	defaultRateLimitPerSecond = 10
	defaultRateLimitBurst     = 20
	sessionTTL                = 24 * time.Hour
	defaultDBMaxConns         = 10 // flash-sale workers (default 4) stay well below it: the outbox, the janitor and reads share the pool
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
	_ service.ExpiryStore      = (*repository.Expiry)(nil)
	_ service.StockGate        = (*repository.Reservations)(nil)
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
	// The wrapper adds trace_id and span_id to any record logged with a context that carries a span.
	logger := slog.New(telemetry.NewLogHandler(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(logger); err != nil {
		logger.Error("api stopped", "err", err)
		os.Exit(1)
	}
}

// run returns errors instead of calling os.Exit, so every defer (closing the pool) actually runs.
func run(logger *slog.Logger) error {
	// Ctrl+C (SIGINT) and docker stop (SIGTERM) both cancel ctx. ctx only means "stop taking requests": the background
	// goroutines below deliberately do not hang off it, because running requests may still need them (see serveThenStop).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Telemetry first, so even the startup logs and every later span have a home. OTEL_TRACES_EXPORTER=stdout prints spans
	// to stderr (stdout is the JSON log); unset or "none" keeps trace ids in the logs without exporting anything.
	tp, err := telemetry.NewTracerProvider(os.Getenv("OTEL_TRACES_EXPORTER"), os.Stderr)
	if err != nil {
		return err
	}
	otel.SetTracerProvider(tp)
	defer func() {
		flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetry.Shutdown(flushCtx, tp); err != nil {
			logger.Error("flush spans", "err", err)
		}
	}()
	metrics := telemetry.NewMetrics()

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

	// The pool has a ceiling on purpose: a stampede must wait in our queue, not pile up as connections in PostgreSQL.
	pool, err := postgres.OpenWithMaxConns(ctx, dbURL, int32(envInt("DB_MAX_CONNS", defaultDBMaxConns)))
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
		handler.Instrument(metrics), // before the limiter, so a 429 is counted too
		handler.RateLimit(limiter, logger),
		handler.Session(sessions, sessionTTL, logger),
	)

	verifier, err := ecpayVerifier(logger)
	if err != nil {
		return err
	}
	payments := service.NewPaymentService(verifier, repository.NewPayments(pool), logger)

	// Background goroutines live in one group with its own context, not ctx: a signal must not stop them while requests
	// are still draining. The deferred Stop is for the early returns below; the normal path stops them in serveThenStop,
	// right after the server has drained. Registered after pool.Close, so it runs before it: nothing touches a closed pool.
	workers := lifecycle.NewGroup()
	defer workers.Stop()

	dispatcher := service.NewOutboxDispatcher(repository.NewOutbox(pool), logMailer{logger}, service.OutboxConfig{}, logger)
	workers.Go(dispatcher.Run)
	janitor := service.NewExpiryJanitor(repository.NewExpiry(pool), service.ExpiryConfig{}, logger)
	workers.Go(janitor.Run)

	// Flash sale: switched on by FLASH_SALE_PRODUCT_ID and FLASH_SALE_STOCK. Off by default, the route does not exist.
	var apiHandler http.Handler = h
	if flashProduct := envInt("FLASH_SALE_PRODUCT_ID", 0); flashProduct > 0 {
		reservations := repository.NewReservations(rdb)
		created, err := reservations.Open(ctx, flashProduct, envInt("FLASH_SALE_STOCK", 1))
		if err != nil {
			return err
		}
		sale := service.NewFlashSale(reservations, repository.NewOrders(pool),
			service.FlashConfig{Workers: envInt("FLASH_SALE_WORKERS", 0), QueueSize: envInt("FLASH_SALE_QUEUE", 0)}, logger)
		workers.Go(sale.Run) // the third goroutine: the flash-sale workers
		apiHandler = handler.WithFlashSale(h, sale, logger, handler.Instrument(metrics), handler.RateLimit(limiter, logger))
		logger.Info("flash sale open", "product", flashProduct, "counter_created", created)
	}

	// otelhttp is the outermost layer: it opens the request's span before anything else runs, so RequestLog and every
	// handler below see a context that already carries the trace id. /metrics sits beside the shop, outside the rate
	// limiter and the session, because Prometheus is not a visitor. Real deployments scrape it on an internal port.
	shop := otelhttp.NewHandler(handler.WithPaymentCallback(apiHandler, payments, logger, handler.Instrument(metrics)), "shop", otelhttp.WithTracerProvider(tp))
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", metrics.Handler())
	mux.Handle("/", shop)
	srv := &http.Server{Handler: mux}
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	logger.Info("api listening", "port", port)
	// After the first signal, give Ctrl+C back its default meaning: a second one then ends the process even mid-drain.
	go func() { <-ctx.Done(); stop() }()

	// Blocks until a signal arrives, then: refuse new connections, let running requests finish, stop the workers.
	// The defers after this return close Redis and the pool, and flush the last spans.
	err = serveThenStop(ctx, srv, ln, time.Duration(envInt("SHUTDOWN_GRACE_SECONDS", int(defaultShutdownGrace/time.Second)))*time.Second, workers)
	if err == nil {
		logger.Info("stop signal received: requests drained, workers stopped")
	}
	return err // a grace period that ran out is an error, so the process exits non-zero and the deploy sees it
}
