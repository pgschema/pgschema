CREATE INDEX IF NOT EXISTS idx_active_orders_customer_date ON orders (customer_id, order_date DESC, total_amount) WHERE status IN ('pending', 'processing', 'confirmed');
