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
    customer_id integer NOT NULL,
    amount numeric(10,2) CHECK (amount > 0),
    created_at timestamptz DEFAULT now() NOT NULL
);

