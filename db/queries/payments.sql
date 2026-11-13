-- name: InsertPayment :execrows
-- ON CONFLICT DO NOTHING turns "this callback was already recorded" into 0 affected rows instead of an error.
INSERT INTO payments (order_id, trade_no, amount_twd) VALUES ($1, $2, $3)
ON CONFLICT (trade_no) DO NOTHING;

-- name: MarkOrderPaid :execrows
-- Matching on the amount too means a callback for the wrong sum marks nothing.
UPDATE orders SET paid_at = COALESCE(paid_at, now()) WHERE id = $1 AND total_twd = $2;
