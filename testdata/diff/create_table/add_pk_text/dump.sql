--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: countries; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS countries (
    code text PRIMARY KEY,
    name text NOT NULL,
    continent text
);

