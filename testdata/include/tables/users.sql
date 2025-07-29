CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL,
    name TEXT NOT NULL
);

ALTER TABLE users ADD CONSTRAINT users_email_check CHECK (email LIKE '%@%');

CREATE INDEX idx_users_email ON users(email);

CREATE INDEX idx_users_name ON users(name);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;

CREATE POLICY users_policy ON users FOR ALL USING (true);

COMMENT ON TABLE users IS 'User accounts';

COMMENT ON COLUMN users.email IS 'User email address';

CREATE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_update_trigger
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();