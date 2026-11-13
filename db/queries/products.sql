-- name: ListProducts :many
SELECT id, name, price_twd
FROM products
ORDER BY id;

-- name: LockProduct :one
-- FOR UPDATE: the row lock is held until the surrounding transaction ends. Other transactions that ask for
-- the same row wait here, so "read the stock, then decide" cannot interleave with another buyer.
SELECT id, price_twd, stock
FROM products
WHERE id = $1
FOR UPDATE;

-- name: DecreaseStock :exec
UPDATE products SET stock = stock - $2 WHERE id = $1;

-- name: IncreaseStock :exec
UPDATE products SET stock = stock + $2 WHERE id = $1;
