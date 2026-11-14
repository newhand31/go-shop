package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	goredis "github.com/redis/go-redis/v9"
)

// ErrSaleNotOpen means nobody opened a flash sale for this product, so there is no counter to take from.
var ErrSaleNotOpen = errors.New("flash sale not open")

// reserveScript is the whole trick of this episode, and it is five lines of Lua. Redis runs a script as one
// uninterrupted step: no other client can run a command between our GET and our DECRBY. Doing the same from Go with
// two commands ("read the counter, decide, write it back") is the oversell race again, one network hop wider.
//
// Returns -1 when the counter does not exist, 0 when there is not enough left, 1 when the units were taken.
var reserveScript = goredis.NewScript(`
local left = redis.call('GET', KEYS[1])
if not left then return -1 end
if tonumber(left) < tonumber(ARGV[1]) then return 0 end
redis.call('DECRBY', KEYS[1], ARGV[1])
return 1`)

// Reservations is a stock counter in Redis, one per product on sale. It is a gate in front of the database, not the
// source of truth: PostgreSQL still decides whether an order exists. The gate only makes sure that, when 10,000 people
// want 50 units, 9,950 of them are told "sold out" without ever touching the database.
type Reservations struct {
	rdb *goredis.Client
}

// NewReservations wraps a Redis client.
func NewReservations(rdb *goredis.Client) *Reservations { return &Reservations{rdb: rdb} }

func saleKey(productID int) string { return "flash:stock:" + strconv.Itoa(productID) }

// Open starts a sale with `stock` units. It uses SET NX, so a restart in the middle of a sale does not put the
// counter back to full. It reports whether this call created the counter.
func (r *Reservations) Open(ctx context.Context, productID, stock int) (created bool, err error) {
	created, err = r.rdb.SetNX(ctx, saleKey(productID), stock, 0).Result()
	if err != nil {
		return false, fmt.Errorf("open flash sale: %w", err)
	}
	return created, nil
}

// Reserve takes qty units from the counter if there are that many. ok=false means sold out.
func (r *Reservations) Reserve(ctx context.Context, productID, qty int) (ok bool, err error) {
	res, err := reserveScript.Run(ctx, r.rdb, []string{saleKey(productID)}, qty).Int()
	if err != nil {
		return false, fmt.Errorf("reserve: %w", err)
	}
	if res < 0 {
		return false, fmt.Errorf("product %d: %w", productID, ErrSaleNotOpen)
	}
	return res == 1, nil
}

// Release gives units back: the compensation for a reservation whose order never happened.
func (r *Reservations) Release(ctx context.Context, productID, qty int) error {
	if err := r.rdb.IncrBy(ctx, saleKey(productID), int64(qty)).Err(); err != nil {
		return fmt.Errorf("release: %w", err)
	}
	return nil
}

// Remaining reads the counter, for tests and dashboards.
func (r *Reservations) Remaining(ctx context.Context, productID int) (int, error) {
	n, err := r.rdb.Get(ctx, saleKey(productID)).Int()
	if errors.Is(err, goredis.Nil) {
		return 0, ErrSaleNotOpen
	}
	if err != nil {
		return 0, fmt.Errorf("read remaining: %w", err)
	}
	return n, nil
}
