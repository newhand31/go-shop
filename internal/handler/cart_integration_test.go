//go:build integration

package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"

	"github.com/newhand31/go-shop/internal/api"
	"github.com/newhand31/go-shop/internal/handler"
	"github.com/newhand31/go-shop/internal/platform/ratelimit"
	"github.com/newhand31/go-shop/internal/platform/redis"
	"github.com/newhand31/go-shop/internal/platform/session"
	"github.com/newhand31/go-shop/internal/repository"
	"github.com/newhand31/go-shop/internal/service"
)

// startRedisURL runs a throwaway Redis container and returns its URL.
func startRedisURL(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	ctr, err := tcredis.Run(ctx, "redis:7-alpine")
	testcontainers.CleanupContainer(t, ctr)
	if err != nil {
		t.Fatalf("start redis container: %v", err)
	}
	url, err := ctr.ConnectionString(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return url
}

type catalog struct{}

func (catalog) List(context.Context) ([]service.Product, error) {
	return []service.Product{{ID: 1, Name: "keyboard", PriceTWD: 1500}, {ID: 2, Name: "mouse", PriceTWD: 500}}, nil
}

// memCarts is the episode-4 way of keeping state: a map inside one process. Test-only, to replay the incident.
type memCarts struct {
	mu    sync.Mutex
	carts map[string]map[int]int
}

func (m *memCarts) Items(_ context.Context, id string) (map[int]int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[int]int{}
	for k, v := range m.carts[id] {
		out[k] = v
	}
	return out, nil
}

func (m *memCarts) SetItem(_ context.Context, id string, p, q int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.carts == nil {
		m.carts = map[string]map[int]int{}
	}
	if m.carts[id] == nil {
		m.carts[id] = map[int]int{}
	}
	m.carts[id][p] = q
	return nil
}

func (m *memCarts) RemoveItem(_ context.Context, id string, p int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.carts[id], p)
	return nil
}

// newAPI builds one "API instance": its own router, limiter and middleware, over the given stores.
// Two calls give two instances, like two machines behind a load balancer.
func newAPI(t *testing.T, sessions handler.SessionStore, carts service.CartStore) *httptest.Server {
	t.Helper()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	ttl := time.Hour
	h := handler.New(oneProduct{}, service.NewCartService(carts, catalog{}), logger,
		handler.RateLimit(ratelimit.New(1000, 1000), logger),
		handler.Session(sessions, ttl, logger),
	)
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

// browser is an HTTP client with a cookie jar, and it remembers which server to talk to.
type browser struct {
	t      *testing.T
	client *http.Client
}

func newBrowser(t *testing.T) *browser {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &browser{t: t, client: &http.Client{Jar: jar}}
}

func (b *browser) do(method, url, body string) api.Cart {
	b.t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		b.t.Fatal(err)
	}
	resp, err := b.client.Do(req)
	if err != nil {
		b.t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		b.t.Fatalf("%s %s = %d: %s", method, url, resp.StatusCode, raw)
	}
	var cart api.Cart
	if err := json.NewDecoder(resp.Body).Decode(&cart); err != nil {
		b.t.Fatal(err)
	}
	return cart
}

// The incident, replayed: the cart lives in the memory of machine A, the customer lands on machine B.
// (A cookie jar keys cookies by host, not port, so one jar can talk to both servers like a real browser would.)
func TestCartIsLostWhenStateLivesInProcessMemory(t *testing.T) {
	a := newAPI(t, session.NewMemory(time.Hour), &memCarts{})
	b := newAPI(t, session.NewMemory(time.Hour), &memCarts{})
	me := newBrowser(t)

	me.do(http.MethodPut, a.URL+"/cart/items/1", `{"quantity":3}`)
	if got := me.do(http.MethodGet, a.URL+"/cart", ""); len(got.Items) != 1 {
		t.Fatalf("machine A: %+v, want the item", got)
	}
	if got := me.do(http.MethodGet, b.URL+"/cart", ""); len(got.Items) != 0 {
		t.Fatalf("machine B: %+v, want an empty cart (that is the bug)", got)
	}
}

// The fix: same two machines, but session and cart live in Redis.
func TestCartSurvivesSwitchingMachinesWithRedis(t *testing.T) {
	ctx := context.Background()
	rdb, err := redis.Open(ctx, startRedisURL(t))
	if err != nil {
		t.Fatal(err)
	}
	defer rdb.Close()

	// Each instance has its own Redis-backed objects, as separate processes would.
	a := newAPI(t, session.NewRedis(rdb, time.Hour), repository.NewCarts(rdb))
	b := newAPI(t, session.NewRedis(rdb, time.Hour), repository.NewCarts(rdb))
	me := newBrowser(t)

	me.do(http.MethodPut, a.URL+"/cart/items/1", `{"quantity":3}`)
	me.do(http.MethodPut, a.URL+"/cart/items/2", `{"quantity":1}`)

	got := me.do(http.MethodGet, b.URL+"/cart", "")
	if len(got.Items) != 2 || got.TotalTwd != 3*1500+500 {
		t.Fatalf("machine B sees %+v, want both items and total 5000", got)
	}

	got = me.do(http.MethodDelete, b.URL+"/cart/items/1", "")
	if len(got.Items) != 1 || got.Items[0].ProductId != 2 {
		t.Fatalf("after delete on B: %+v", got)
	}
	if got := me.do(http.MethodGet, a.URL+"/cart", ""); len(got.Items) != 1 {
		t.Fatalf("machine A after B deleted: %+v", got)
	}

	// A different browser must not see this cart.
	other := newBrowser(t)
	if got := other.do(http.MethodGet, a.URL+"/cart", ""); len(got.Items) != 0 {
		t.Fatalf("a stranger sees %+v, want an empty cart", got)
	}
}

// Carts and sessions both expire in Redis, and every write or use pushes the expiry forward.
func TestRedisExpiry(t *testing.T) {
	ctx := context.Background()
	rdb, err := redis.Open(ctx, startRedisURL(t))
	if err != nil {
		t.Fatal(err)
	}
	defer rdb.Close()

	sessions := session.NewRedis(rdb, 2*time.Second)
	id, err := sessions.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := sessions.Exists(ctx, id); err != nil || !ok {
		t.Fatalf("fresh session: %v, %v", ok, err)
	}
	if ok, _ := sessions.Exists(ctx, "never-created"); ok {
		t.Fatal("unknown session reported as alive")
	}
	time.Sleep(2500 * time.Millisecond)
	if ok, _ := sessions.Exists(ctx, id); ok {
		t.Fatal("session should have expired")
	}

	carts := repository.NewCarts(rdb)
	if err := carts.SetItem(ctx, "c1", 1, 2); err != nil {
		t.Fatal(err)
	}
	ttl, err := rdb.TTL(ctx, "cart:c1").Result()
	if err != nil || ttl <= 0 || ttl > 7*24*time.Hour {
		t.Fatalf("cart ttl = %v, %v; want a positive expiry of at most 7 days", ttl, err)
	}
	items, err := carts.Items(ctx, "c1")
	if err != nil || items[1] != 2 {
		t.Fatalf("items = %v, %v", items, err)
	}
	if items, err := carts.Items(ctx, "nobody"); err != nil || len(items) != 0 {
		t.Fatalf("unknown cart = %v, %v; want empty, no error", items, err)
	}
}
