CREATE POLICY "UserPolicy" ON users TO PUBLIC USING (tenant_id = current_setting('app.current_tenant')::integer);

CREATE POLICY "my-policy" ON users FOR INSERT TO PUBLIC WITH CHECK ((role)::text = 'user');

CREATE POLICY "select" ON users FOR SELECT TO PUBLIC USING (true);

CREATE POLICY user_tenant_isolation ON users FOR UPDATE TO PUBLIC USING (tenant_id = current_setting('app.current_tenant')::integer);
