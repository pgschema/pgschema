CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    status TEXT NOT NULL DEFAULT 'pending',
    amount DECIMAL(10,2) DEFAULT 0.00
);

ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (status IN ('pending', 'completed'));

CREATE INDEX idx_orders_user_id ON orders(user_id);

CREATE INDEX idx_orders_status ON orders(status);

ALTER TABLE orders ENABLE ROW LEVEL SECURITY;

CREATE POLICY orders_policy ON orders FOR ALL USING (user_id = 1);

COMMENT ON TABLE orders IS 'Customer orders';

COMMENT ON COLUMN orders.user_id IS 'Reference to user';