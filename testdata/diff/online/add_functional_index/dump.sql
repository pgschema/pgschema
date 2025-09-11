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
    first_name text,
    last_name text,
    email text,
    phone text,
    created_at timestamptz,
    status text
);

--
-- Name: idx_users_fullname_search; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_users_fullname_search ON users (lower(first_name), lower(last_name), lower(email));

