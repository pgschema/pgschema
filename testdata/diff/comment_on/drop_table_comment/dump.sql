--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: inventory; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS inventory (
    id SERIAL PRIMARY KEY,
    product_id integer NOT NULL,
    quantity integer NOT NULL
);

