--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.4.0


--
-- Name: users; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS users (
    id bigint,
    email varchar(255) NOT NULL,
    last_name varchar(100),
    department varchar(50),
    status varchar(20),
    created_at timestamptz DEFAULT now(),
    CONSTRAINT users_pkey PRIMARY KEY (id)
);

--
-- Name: idx_users_created_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_users_created_at ON users (created_at);

--
-- Name: idx_users_department; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_users_department ON users (department);

--
-- Name: idx_users_email; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

--
-- Name: idx_users_last_name; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_users_last_name ON users (last_name);

--
-- Name: idx_users_status; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_users_status ON users (status);

