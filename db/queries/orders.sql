-- name: InsertOrder :one
INSERT INTO orders (total_twd) VALUES ($1) RETURNING id;

-- name: InsertOrderLine :exec
INSERT INTO order_lines (order_id, product_id, quantity, unit_price_twd) VALUES ($1, $2, $3, $4);
