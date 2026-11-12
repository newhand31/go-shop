-- name: ListProducts :many
SELECT id, name, price_twd
FROM products
ORDER BY id;
