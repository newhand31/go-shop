-- +goose Up
-- Stock lives next to the product. The CHECK is the last line of defence: even buggy code cannot store a negative number.
ALTER TABLE products ADD COLUMN stock integer NOT NULL DEFAULT 0 CHECK (stock >= 0);
-- The monitor is the incident product: exactly one left.
UPDATE products SET stock = CASE id WHEN 3 THEN 1 ELSE 10 END;

CREATE TABLE orders (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    total_twd  integer     NOT NULL CHECK (total_twd >= 0),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE order_lines (
    order_id       bigint  NOT NULL REFERENCES orders (id),
    product_id     integer NOT NULL REFERENCES products (id),
    quantity       integer NOT NULL CHECK (quantity > 0),
    unit_price_twd integer NOT NULL CHECK (unit_price_twd >= 0),
    PRIMARY KEY (order_id, product_id)
);

-- +goose Down
DROP TABLE order_lines;
DROP TABLE orders;
ALTER TABLE products DROP COLUMN stock;
