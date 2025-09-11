--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: products; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS products (
    id integer NOT NULL,
    name text,
    price numeric(10,2),
    category_id integer
);

