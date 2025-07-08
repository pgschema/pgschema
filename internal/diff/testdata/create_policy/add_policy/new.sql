CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    tenant_id INTEGER NOT NULL
);

-- RLS is enabled with new policy
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

CREATE POLICY user_tenant_isolation ON users
    FOR ALL
    TO PUBLIC
    USING (tenant_id = current_setting('app.current_tenant')::INTEGER);