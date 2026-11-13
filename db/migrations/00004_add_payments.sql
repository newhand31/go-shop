-- +goose Up
-- An order is paid once ECPay's callback says so.
ALTER TABLE orders ADD COLUMN paid_at timestamptz;

-- One row per payment ECPay reports. trade_no is ECPay's own transaction number and is the same on every resend of
-- the same callback, so the UNIQUE constraint is the idempotency key: the database, not application code, decides
-- that a payment is only recorded once.
CREATE TABLE payments (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id   bigint      NOT NULL REFERENCES orders (id),
    trade_no   text        NOT NULL UNIQUE,
    amount_twd integer     NOT NULL CHECK (amount_twd >= 0),
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE payments;
ALTER TABLE orders DROP COLUMN paid_at;
