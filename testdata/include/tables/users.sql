--
-- Name: users; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE users (
    id integer PRIMARY KEY,
    email text NOT NULL CHECK (email LIKE '%@%'),
    name text NOT NULL
);

COMMENT ON TABLE users IS 'User accounts';

COMMENT ON COLUMN users.email IS 'User email address';

--
-- Name: idx_users_email; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_users_email ON users (email);

--
-- Name: idx_users_name; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_users_name ON users (name);

--
-- Name: users; Type: TABLE; Schema: -; Owner: -
--

ALTER TABLE users ENABLE ROW LEVEL SECURITY;

--
-- Name: users_policy; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY users_policy ON users TO PUBLIC USING (true);

--
-- Name: users_update_trigger; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER users_update_trigger
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();