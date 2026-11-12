//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/newhand31/go-shop/internal/platform/postgres"
	"github.com/newhand31/go-shop/internal/repository"
)

// startDB runs a throwaway PostgreSQL container, applies the migrations and returns a connection URL.
func startDB(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	ctr, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("goshop"),
		tcpostgres.WithUsername("goshop"),
		tcpostgres.WithPassword("goshop"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	testcontainers.CleanupContainer(t, ctr)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	url, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	return url
}

func TestProducts_List_ReturnsSeededProducts(t *testing.T) {
	ctx := context.Background()
	url := startDB(t)

	pool, err := postgres.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := postgres.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}

	got, err := repository.NewProducts(pool).List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := []repository.Product{
		{ID: 1, Name: "機械式鍵盤", PriceTWD: 2490},
		{ID: 2, Name: "人體工學滑鼠", PriceTWD: 1290},
		{ID: 3, Name: "27 吋螢幕", PriceTWD: 5990},
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%+v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// The episode's incident: products must survive a restart. A new pool stands in for a restarted process.
func TestProducts_SurviveReconnect(t *testing.T) {
	ctx := context.Background()
	url := startDB(t)

	first, err := postgres.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	if err := postgres.Migrate(ctx, first); err != nil {
		t.Fatal(err)
	}
	if _, err := first.Exec(ctx, "INSERT INTO products (name, price_twd) VALUES ($1, $2)", "USB-C hub", 890); err != nil {
		t.Fatalf("insert: %v", err)
	}
	first.Close()

	second, err := postgres.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	// Running migrations again must be a no-op, the same as every restart of the API.
	if err := postgres.Migrate(ctx, second); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	got, err := repository.NewProducts(second).List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 4 || got[3].Name != "USB-C hub" {
		t.Fatalf("got %+v, want the 3 seeded products plus USB-C hub", got)
	}
}
