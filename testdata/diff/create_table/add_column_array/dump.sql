--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: articles; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS articles (
    id integer NOT NULL,
    title text,
    tags text[]
);

