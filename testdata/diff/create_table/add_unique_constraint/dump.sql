--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: user_sessions; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS user_sessions (
    user_id integer NOT NULL,
    session_token text NOT NULL,
    device_fingerprint text NOT NULL,
    created_at timestamp NOT NULL,
    UNIQUE (session_token, device_fingerprint),
    UNIQUE (user_id, device_fingerprint)
);

