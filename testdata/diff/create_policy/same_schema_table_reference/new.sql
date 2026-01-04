-- Test case for Issue #224: Table references in policy expressions
-- This tests that same-schema table references are properly normalized

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id)
);

ALTER TABLE orders ENABLE ROW LEVEL SECURITY;

-- Policy with subquery referencing another table in the same schema
-- The table reference "users" should be normalized regardless of schema prefix
CREATE POLICY select_own_orders ON orders
    FOR SELECT
    TO PUBLIC
    USING (user_id IN (SELECT u.id FROM users u WHERE u.tenant_id = 1));
