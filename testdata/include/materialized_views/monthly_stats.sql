--
-- Name: monthly_stats; Type: MATERIALIZED VIEW; Schema: -; Owner: -
--

CREATE MATERIALIZED VIEW IF NOT EXISTS monthly_stats AS
 SELECT date_trunc('month'::text, now()) AS month,
    count(user_id) AS unique_customers,
    count(id) AS total_orders,
    sum(amount) AS total_revenue,
    avg(amount) AS avg_order_value,
    count(
        CASE
            WHEN status = 'completed'::text THEN 1
            ELSE NULL::integer
        END) AS completed_orders,
    count(
        CASE
            WHEN status = 'pending'::text THEN 1
            ELSE NULL::integer
        END) AS pending_orders
   FROM orders o
  GROUP BY (date_trunc('month'::text, now()));

COMMENT ON MATERIALIZED VIEW monthly_stats IS 'Monthly order statistics';

--
-- Name: idx_monthly_stats_month; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_monthly_stats_month ON monthly_stats (month);

--
-- Name: idx_monthly_stats_revenue; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_monthly_stats_revenue ON monthly_stats (total_revenue DESC);
