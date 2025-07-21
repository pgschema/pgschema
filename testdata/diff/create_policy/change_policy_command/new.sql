CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    tenant_id INTEGER NOT NULL
);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Changed command from ALL to SELECT (requires recreation)
CREATE POLICY user_tenant_isolation ON users
    FOR SELECT
    TO PUBLIC
    USING (tenant_id = 1);