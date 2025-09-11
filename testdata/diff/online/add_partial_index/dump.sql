--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: orders; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS orders (
    id integer NOT NULL,
    customer_id integer,
    status text,
    order_date date,
    total_amount numeric(10,2),
    payment_status text,
    created_at timestamptz,
    updated_at timestamptz
);

--
-- Name: idx_active_orders_customer_date; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_active_orders_customer_date ON orders (customer_id, order_date DESC, total_amount) WHERE status IN ('pending', 'processing', 'confirmed');

