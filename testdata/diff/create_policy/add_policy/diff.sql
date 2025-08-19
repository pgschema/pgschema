CREATE POLICY user_tenant_isolation ON users TO PUBLIC USING (tenant_id = current_setting('app.current_tenant')::integer);
