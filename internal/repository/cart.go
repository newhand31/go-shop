package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// cartTTL is how long an untouched cart survives. Every write pushes it forward again.
const cartTTL = 7 * 24 * time.Hour

// Carts keeps one Redis hash per cart: key "cart:<id>", field = product id, value = quantity.
//
// Note what is missing: there is no "implements CartStore" anywhere in this file. The service package declares the
// interface it needs; this type just happens to have methods that fit. Go checks the fit where the two meet (main).
type Carts struct {
	rdb *goredis.Client
}

// NewCarts wraps a Redis client.
func NewCarts(rdb *goredis.Client) *Carts { return &Carts{rdb: rdb} }

func cartKey(cartID string) string { return "cart:" + cartID }

// Items returns product id → quantity. An unknown cart is an empty cart, not an error.
func (c *Carts) Items(ctx context.Context, cartID string) (map[int]int, error) {
	raw, err := c.rdb.HGetAll(ctx, cartKey(cartID)).Result()
	if err != nil {
		return nil, fmt.Errorf("read cart: %w", err)
	}
	items := make(map[int]int, len(raw))
	for field, value := range raw {
		productID, err1 := strconv.Atoi(field)
		quantity, err2 := strconv.Atoi(value)
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("read cart: corrupt entry %q=%q", field, value)
		}
		items[productID] = quantity
	}
	return items, nil
}

// SetItem sets the quantity of one product and refreshes the cart's expiry. The two commands go out in one round trip.
func (c *Carts) SetItem(ctx context.Context, cartID string, productID, quantity int) error {
	key := cartKey(cartID)
	_, err := c.rdb.Pipelined(ctx, func(p goredis.Pipeliner) error {
		p.HSet(ctx, key, strconv.Itoa(productID), quantity)
		p.Expire(ctx, key, cartTTL)
		return nil
	})
	if err != nil {
		return fmt.Errorf("set cart item: %w", err)
	}
	return nil
}

// RemoveItem deletes one product from the cart; removing something that is not there is not an error.
func (c *Carts) RemoveItem(ctx context.Context, cartID string, productID int) error {
	if err := c.rdb.HDel(ctx, cartKey(cartID), strconv.Itoa(productID)).Err(); err != nil {
		return fmt.Errorf("remove cart item: %w", err)
	}
	return nil
}
