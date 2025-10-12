--
-- Name: user_order_summary; Type: MATERIALIZED VIEW; Schema: -; Owner: -
--

CREATE MATERIALIZED VIEW IF NOT EXISTS user_order_summary AS
 SELECT u.id AS user_id,
    u.name,
    u.email,
    count(o.id) AS total_orders,
    sum(o.amount) AS total_amount,
    max(o.amount) AS max_order_amount,
    avg(o.amount) AS avg_order_amount
   FROM users u
     LEFT JOIN orders o ON u.id = o.user_id
  GROUP BY u.id, u.name, u.email;

COMMENT ON MATERIALIZED VIEW user_order_summary IS 'Aggregated user order statistics';

--
-- Name: idx_user_order_summary_total_amount; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_user_order_summary_total_amount ON user_order_summary (total_amount DESC);

--
-- Name: idx_user_order_summary_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_order_summary_user_id ON user_order_summary (user_id);
