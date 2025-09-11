--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: transactions; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS transactions (
    id bigint UNIQUE,
    amount numeric(15,2) NOT NULL,
    created_at timestamptz
);

