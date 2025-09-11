--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: user_permissions; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS user_permissions (
    user_id integer,
    resource_id integer,
    permission_type text,
    granted_at timestamptz,
    PRIMARY KEY (user_id, resource_id, permission_type)
);

