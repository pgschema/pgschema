CREATE TABLE IF NOT EXISTS products (
    id SERIAL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    deleted_at timestamptz,
    CONSTRAINT products_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    deleted_at timestamptz,
    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT users_check CHECK (created_at <= updated_at)
);

COMMENT ON TABLE users IS 'Template for timestamp fields';

CREATE INDEX IF NOT EXISTS users_created_at_idx ON users (created_at);

COMMENT ON COLUMN _template_timestamps.created_at IS NULL;
