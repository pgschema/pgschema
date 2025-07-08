DROP POLICY IF EXISTS user_tenant_isolation ON users;
CREATE POLICY user_tenant_isolation ON users FOR SELECT TO PUBLIC USING ((tenant_id = 1));