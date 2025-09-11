--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: user_pending_permissions; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS user_pending_permissions (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    permission text NOT NULL,
    object_ids_ints bigint[]
);

