ALTER TABLE users ENABLE ROW LEVEL SECURITY;

CREATE POLICY user_tenant_isolation ON users TO PUBLIC USING (tenant_id = 1);