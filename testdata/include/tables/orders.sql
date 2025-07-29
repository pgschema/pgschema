--
-- Name: orders; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE orders (
    id integer NOT NULL PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id),
    status text NOT NULL DEFAULT 'pending',
    amount numeric(10,2) DEFAULT 0.00
);

ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (status IN ('pending', 'completed'));

CREATE INDEX idx_orders_user_id ON orders(user_id);

CREATE INDEX idx_orders_status ON orders(status);

ALTER TABLE orders ENABLE ROW LEVEL SECURITY;

--
-- Name: orders_policy; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY orders_policy ON orders TO PUBLIC USING (user_id = 1);

COMMENT ON TABLE orders IS 'Customer orders';

COMMENT ON COLUMN orders.user_id IS 'Reference to user';