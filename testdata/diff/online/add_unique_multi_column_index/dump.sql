--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: user_profiles; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS user_profiles (
    id integer NOT NULL,
    user_id integer,
    email text,
    username text,
    organization_id integer,
    created_at timestamptz,
    deleted_at timestamptz
);

--
-- Name: idx_unique_email_org; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_email_org ON user_profiles (email, organization_id) WHERE (deleted_at IS NULL);

