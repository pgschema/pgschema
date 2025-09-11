--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: user_permissions; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS user_permissions (
    user_id integer NOT NULL,
    resource_id integer NOT NULL,
    permission_type text NOT NULL,
    granted_at timestamptz DEFAULT now(),
    UNIQUE (user_id, resource_id, permission_type)
);

