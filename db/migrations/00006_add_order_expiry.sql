-- +goose Up
-- An order that was never paid holds its stock. Once the janitor gives that stock back it stamps expired_at, so the same
-- order is never released twice.
ALTER TABLE orders ADD COLUMN expired_at timestamptz;

-- The janitor only looks at orders nobody paid and nobody released.
CREATE INDEX orders_unpaid_idx ON orders (created_at) WHERE paid_at IS NULL AND expired_at IS NULL;

-- +goose Down
DROP INDEX orders_unpaid_idx;
ALTER TABLE orders DROP COLUMN expired_at;
