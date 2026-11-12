-- +goose Up
-- The same three products episode 1 kept in memory.
INSERT INTO products (name, price_twd) VALUES
    ('機械式鍵盤', 2490),
    ('人體工學滑鼠', 1290),
    ('27 吋螢幕', 5990);

-- +goose Down
DELETE FROM products WHERE name IN ('機械式鍵盤', '人體工學滑鼠', '27 吋螢幕');
