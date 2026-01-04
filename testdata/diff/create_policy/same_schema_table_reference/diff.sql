CREATE POLICY select_own_orders ON orders FOR SELECT TO PUBLIC USING (user_id IN ( SELECT u.id FROM users u WHERE (u.tenant_id = 1)));
