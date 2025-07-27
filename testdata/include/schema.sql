-- Main schema file demonstrating \i include functionality
-- This represents a modular approach to organizing database schema
-- Includes ALL supported PostgreSQL database objects

-- Include custom types first (dependencies for tables)
CREATE TYPE user_status AS ENUM ('active', 'inactive');

CREATE TYPE order_status AS ENUM ('pending', 'completed');

CREATE TYPE address AS (
    street TEXT,
    city TEXT
);

-- Include domain types (constrained base types)
CREATE DOMAIN email_address AS TEXT
    CHECK (VALUE LIKE '%@%');

CREATE DOMAIN positive_integer AS INTEGER
    CHECK (VALUE > 0);

-- Include sequences (may be used by tables)  
CREATE SEQUENCE global_id_seq START WITH 1000;

CREATE SEQUENCE order_number_seq START WITH 100000;

-- Include core tables (with their constraints, indexes, and policies)
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL,
    name TEXT NOT NULL
);

ALTER TABLE users ADD CONSTRAINT users_email_check CHECK (email LIKE '%@%');

CREATE INDEX idx_users_email ON users(email);

CREATE INDEX idx_users_name ON users(name);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;

CREATE POLICY users_policy ON users FOR ALL USING (true);

COMMENT ON TABLE users IS 'User accounts';

COMMENT ON COLUMN users.email IS 'User email address';
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

-- Include functions and procedures
CREATE FUNCTION get_user_count()
RETURNS INTEGER AS $$
    SELECT COUNT(*) FROM users;
$$ LANGUAGE SQL;

CREATE FUNCTION get_order_count(user_id_param INTEGER)
RETURNS INTEGER AS $$
    SELECT COUNT(*) FROM orders WHERE user_id = user_id_param;
$$ LANGUAGE SQL;
CREATE PROCEDURE cleanup_orders()
LANGUAGE SQL
AS $$
    DELETE FROM orders WHERE status = 'completed';
$$;

CREATE PROCEDURE update_status(user_id_param INTEGER, new_status TEXT)
LANGUAGE SQL
AS $$
    UPDATE orders SET status = new_status WHERE user_id = user_id_param;
$$;

-- Include views (depend on tables and functions)
CREATE VIEW user_summary AS
SELECT 
    u.id,
    u.name,
    COUNT(o.id) as order_count
FROM users u
LEFT JOIN orders o ON u.id = o.user_id
GROUP BY u.id, u.name;

CREATE VIEW order_details AS
SELECT 
    o.id,
    o.status,
    u.name as user_name
FROM orders o
JOIN users u ON o.user_id = u.id;

COMMENT ON VIEW user_summary IS 'User order summary';

COMMENT ON VIEW order_details IS 'Order details with user info';

-- Include triggers (depend on tables and functions)
CREATE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_update_trigger
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

-- Add some additional schema directly in main file to test mixed content
CREATE SEQUENCE inline_test_seq START WITH 5000;