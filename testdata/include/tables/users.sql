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