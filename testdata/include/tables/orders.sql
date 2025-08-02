--
-- Name: orders; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS orders (
    id integer PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id),
    status text DEFAULT 'pending' NOT NULL CHECK (status IN('pending', 'completed')),
    amount numeric(10,2) DEFAULT 0.00
);

COMMENT ON TABLE orders IS 'Customer orders';

COMMENT ON COLUMN orders.user_id IS 'Reference to user';

--
-- Name: idx_orders_status; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);

--
-- Name: idx_orders_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders (user_id);

--
-- Name: orders; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE orders ENABLE ROW LEVEL SECURITY;

--
-- Name: orders_policy; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY orders_policy ON orders TO PUBLIC USING (user_id = 1);