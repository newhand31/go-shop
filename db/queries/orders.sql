-- name: InsertOrder :one
INSERT INTO orders (total_twd) VALUES ($1) RETURNING id;

-- name: InsertOrderLine :exec
INSERT INTO order_lines (order_id, product_id, quantity, unit_price_twd) VALUES ($1, $2, $3, $4);

-- name: ClaimExpiredOrder :one
-- One unpaid, unreleased order older than the cutoff. FOR UPDATE SKIP LOCKED: an order a payment callback is updating
-- right now is skipped this round instead of waited for.
SELECT id FROM orders
WHERE paid_at IS NULL AND expired_at IS NULL AND created_at < $1
ORDER BY id
LIMIT 1
FOR UPDATE SKIP LOCKED;

-- name: ListOrderLines :many
-- Ordered by product id: the same lock order checkout uses, so releasing and buying cannot deadlock.
SELECT product_id, quantity FROM order_lines WHERE order_id = $1 ORDER BY product_id;

-- name: MarkOrderExpired :exec
UPDATE orders SET expired_at = now() WHERE id = $1;
