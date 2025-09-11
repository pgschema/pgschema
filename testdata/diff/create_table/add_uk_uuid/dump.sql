--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: documents; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS documents (
    id uuid UNIQUE,
    title text NOT NULL,
    content text
);

