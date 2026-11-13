-- +goose Up
-- The outbox: "things that must happen because an order was placed", written in the SAME transaction as the order.
-- Either the order and its to-do both exist, or neither does. A background worker reads the rows and does the work
-- (sending the confirmation email) later, and retries until it works.
CREATE TABLE outbox (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    topic           text        NOT NULL,
    payload         jsonb       NOT NULL,
    attempts        integer     NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    sent_at         timestamptz,
    last_error      text,
    created_at      timestamptz NOT NULL DEFAULT now()
);

-- The worker only ever asks for unsent rows that are due, oldest first.
CREATE INDEX outbox_due_idx ON outbox (next_attempt_at, id) WHERE sent_at IS NULL;

-- +goose Down
DROP TABLE outbox;
