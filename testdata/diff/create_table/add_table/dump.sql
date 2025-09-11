--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: departments; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS departments (
    id integer PRIMARY KEY,
    name text NOT NULL
);

