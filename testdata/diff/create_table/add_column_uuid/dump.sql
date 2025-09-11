--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: sessions; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS sessions (
    id integer NOT NULL,
    user_id integer,
    token uuid
);

