--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: customers; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS customers (
    id integer NOT NULL,
    name text,
    phone varchar(20)
);

