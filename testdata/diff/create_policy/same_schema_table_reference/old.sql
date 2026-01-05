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
