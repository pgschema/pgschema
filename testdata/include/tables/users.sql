--
-- Name: users; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE users (
    id integer NOT NULL PRIMARY KEY,
    email text NOT NULL,
    name text NOT NULL
);

ALTER TABLE users ADD CONSTRAINT users_email_check CHECK (email ~~ '%@%');

CREATE INDEX idx_users_email ON users(email);

CREATE INDEX idx_users_name ON users(name);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;

--
-- Name: users_policy; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY users_policy ON users TO PUBLIC USING (true);

COMMENT ON TABLE users IS 'User accounts';

COMMENT ON COLUMN users.email IS 'User email address';

--
-- Name: users_update_trigger; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER users_update_trigger
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();