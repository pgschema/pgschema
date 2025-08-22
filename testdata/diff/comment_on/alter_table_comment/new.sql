CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL,
    total DECIMAL(10,2) NOT NULL
);

COMMENT ON TABLE orders IS 'Customer orders with payment and shipping information';