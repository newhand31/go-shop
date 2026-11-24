//go:build integration

package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/newhand31/go-shop/internal/repository"
)

func TestOrderReader_StatusFollowsTheTimestamps(t *testing.T) {
	pool := openMigrated(t)
	ctx := context.Background()
	placed, err := repository.NewOrders(pool).Place(ctx, []repository.OrderLine{{ProductID: 1, Quantity: 2}})
	if err != nil {
		t.Fatal(err)
	}
	reader := repository.NewOrderReader(pool)

	got, err := reader.Get(ctx, placed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalTWD != placed.TotalTWD || len(got.Lines) != 1 || got.Lines[0].Quantity != 2 || got.PaidAt != nil || got.ExpiredAt != nil {
		t.Fatalf("fresh order = %+v, want the placed one with no timestamps", got)
	}

	if _, err := pool.Exec(ctx, `UPDATE orders SET paid_at = now() WHERE id = $1`, placed.ID); err != nil {
		t.Fatal(err)
	}
	if got, err = reader.Get(ctx, placed.ID); err != nil || got.PaidAt == nil {
		t.Fatalf("after paying: %+v, err %v", got, err)
	}
}

func TestOrderReader_UnknownOrder(t *testing.T) {
	pool := openMigrated(t)
	if _, err := repository.NewOrderReader(pool).Get(context.Background(), 424242); !errors.Is(err, repository.ErrOrderNotFound) {
		t.Fatalf("err = %v, want ErrOrderNotFound", err)
	}
}
