CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    tenant_id INTEGER NOT NULL
);

-- RLS is enabled but not forced
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Add a simple policy
CREATE POLICY tenant_isolation ON users
    FOR ALL
    TO PUBLIC
    USING (tenant_id = current_setting('app.current_tenant')::INTEGER);
