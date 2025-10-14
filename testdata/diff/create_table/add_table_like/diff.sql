CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    deleted_at timestamptz
);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    deleted_at timestamptz,
    CONSTRAINT users_check CHECK (created_at <= updated_at)
);

COMMENT ON TABLE users IS 'Template for timestamp fields';

CREATE INDEX IF NOT EXISTS users_created_at_idx ON users (created_at);