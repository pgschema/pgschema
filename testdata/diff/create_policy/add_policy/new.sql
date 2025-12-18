CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    tenant_id INTEGER NOT NULL,
    role VARCHAR(50) NOT NULL
);

-- RLS is enabled with multiple policies demonstrating quoting scenarios
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Policy with reserved word name (requires quoting)
CREATE POLICY "select" ON users
    FOR SELECT
    TO PUBLIC
    USING (true);

-- Policy with mixed case name (requires quoting to preserve case)
CREATE POLICY "UserPolicy" ON users
    FOR ALL
    TO PUBLIC
    USING (tenant_id = current_setting('app.current_tenant')::INTEGER);

-- Policy with special character in name (requires quoting)
CREATE POLICY "my-policy" ON users
    FOR INSERT
    TO PUBLIC
    WITH CHECK (role = 'user');

-- Policy with regular snake_case name (no quoting needed)
CREATE POLICY user_tenant_isolation ON users
    FOR UPDATE
    TO PUBLIC
    USING (tenant_id = current_setting('app.current_tenant')::INTEGER);
