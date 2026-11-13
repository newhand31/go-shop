//go:build integration

package handler_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/newhand31/go-shop/internal/handler"
	"github.com/newhand31/go-shop/internal/platform/ratelimit"
	"github.com/newhand31/go-shop/internal/platform/session"
	"github.com/newhand31/go-shop/internal/service"
)

type oneProduct struct{}

func (oneProduct) List(context.Context) ([]service.Product, error) {
	return []service.Product{{ID: 1, Name: "keyboard", PriceTWD: 1500}}, nil
}

func (oneProduct) Get(context.Context, int) (service.Product, error) {
	return service.Product{ID: 1, Name: "keyboard", PriceTWD: 1500}, nil
}

// countingStore counts Create calls, to prove a flood never gets as far as the session store.
type countingStore struct {
	*session.Memory
	created atomic.Int64
}

func (c *countingStore) Create(ctx context.Context) (string, error) {
	c.created.Add(1)
	return c.Memory.Create(ctx)
}

// newServer wires the real chain over real TCP: real limiter, real session store, real net/http server.
func newServer(t *testing.T, rate float64, burst int) (*httptest.Server, *countingStore) {
	t.Helper()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	ttl := time.Hour
	store := &countingStore{Memory: session.NewMemory(ttl)}
	h := handler.New(oneProduct{}, noCarts{}, noOrders{}, logger,
		handler.RateLimit(ratelimit.New(rate, burst), logger),
		handler.Session(store, ttl, logger),
	)
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv, store
}

// The incident: one IP, a thousand requests. The limiter must let through only burst + what refilled meanwhile.
func TestFloodFromOneIPIsMostlyRejected(t *testing.T) {
	const burst, rate, total = 10, 5.0, 1000
	srv, store := newServer(t, rate, burst)

	var ok, limited int
	start := time.Now()
	for i := 0; i < total; i++ {
		resp, err := http.Get(srv.URL + "/products")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusOK:
			ok++
		case http.StatusTooManyRequests:
			limited++
			if resp.Header.Get("Retry-After") == "" {
				t.Fatal("429 without Retry-After")
			}
		default:
			t.Fatalf("unexpected status %d", resp.StatusCode)
		}
	}
	elapsed := time.Since(start)

	maxOK := burst + int(rate*elapsed.Seconds()) + 1
	if ok > maxOK {
		t.Fatalf("%d requests passed in %v, at most %d expected", ok, elapsed, maxOK)
	}
	if ok < burst {
		t.Fatalf("only %d requests passed, the burst of %d should", ok, burst)
	}
	if limited == 0 {
		t.Fatal("nothing was rate limited")
	}
	// Every request without a cookie that passes the limiter creates one session; the rejected ones create none.
	if got := store.created.Load(); got > int64(ok) {
		t.Fatalf("%d sessions created for %d passed requests: the flood reached the session store", got, ok)
	}
	t.Logf("ok=%d limited=%d elapsed=%v", ok, limited, elapsed)
}

// Many goroutines share one limiter; the mutex inside must keep the count honest.
func TestLimiterHoldsUnderConcurrency(t *testing.T) {
	const burst, rate = 20, 1.0
	srv, _ := newServer(t, rate, burst)

	var passed atomic.Int64
	var wg sync.WaitGroup
	start := time.Now()
	for g := 0; g < 20; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 25; i++ {
				resp, err := http.Get(srv.URL + "/healthz")
				if err != nil {
					t.Error(err)
					return
				}
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					passed.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	maxOK := int64(burst + int(rate*time.Since(start).Seconds()) + 1)
	if got := passed.Load(); got > maxOK {
		t.Fatalf("%d requests passed, at most %d expected", got, maxOK)
	}
}

// A browser (cookie jar) gets one session id on its first visit and keeps it.
func TestSessionCookieSticks(t *testing.T) {
	srv, _ := newServer(t, 100, 100)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar}

	var setCookies []int
	for i := 0; i < 3; i++ {
		resp, err := client.Get(srv.URL + "/products")
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		setCookies = append(setCookies, len(resp.Cookies()))
	}
	if setCookies[0] != 1 || setCookies[1] != 0 || setCookies[2] != 0 {
		t.Fatalf("Set-Cookie count per response = %v, want [1 0 0]", setCookies)
	}
}

// noCarts is a Carts for tests that only exercise middleware.
type noCarts struct{}

func (noCarts) Get(context.Context, string) (service.Cart, error) { return service.Cart{}, nil }
func (noCarts) SetItem(context.Context, string, int, int) (service.Cart, error) {
	return service.Cart{}, nil
}
func (noCarts) RemoveItem(context.Context, string, int) (service.Cart, error) {
	return service.Cart{}, nil
}

// noOrders is an Orders for tests that do not check out.
type noOrders struct{}

func (noOrders) Checkout(context.Context, string) (service.Order, error) {
	return service.Order{}, nil
}
