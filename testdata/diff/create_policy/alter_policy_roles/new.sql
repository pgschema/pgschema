CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    tenant_id INTEGER NOT NULL
);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Changed the roles from PUBLIC to specific roles
CREATE POLICY user_tenant_isolation ON users
    FOR ALL
    TO testuser
    USING (tenant_id = current_setting('app.current_tenant')::INTEGER);