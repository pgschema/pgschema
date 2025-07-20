DROP POLICY IF EXISTS user_tenant_isolation ON users;

CREATE POLICY tenant_access_policy ON users TO PUBLIC USING ((tenant_id = 1));