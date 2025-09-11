--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: products; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS products (
    name text NOT NULL,
    price numeric(10,2),
    id bigint GENERATED ALWAYS AS IDENTITY NOT NULL
);

