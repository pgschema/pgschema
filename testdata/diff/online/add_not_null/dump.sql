--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: users; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS users (
    id integer NOT NULL,
    username text NOT NULL,
    email text NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL
);

