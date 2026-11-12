-- +goose Up
CREATE TABLE products (
    id         integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       text        NOT NULL,
    price_twd  integer     NOT NULL CHECK (price_twd >= 0),
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE products;
