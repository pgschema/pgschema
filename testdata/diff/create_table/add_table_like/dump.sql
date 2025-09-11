--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: _template_timestamps; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS _template_timestamps (
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    deleted_at timestamptz,
    CHECK (created_at <= updated_at)
);


COMMENT ON TABLE _template_timestamps IS 'Template for timestamp fields';


COMMENT ON COLUMN _template_timestamps.created_at IS 'Record creation time';

--
-- Name: idx_template_created_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_template_created_at ON _template_timestamps (created_at);

--
-- Name: products; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    deleted_at timestamptz
);

--
-- Name: users; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    deleted_at timestamptz,
    CHECK (created_at <= updated_at)
);


COMMENT ON COLUMN users.created_at IS 'Record creation time';

--
-- Name: users_created_at_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS users_created_at_idx ON users (created_at);

