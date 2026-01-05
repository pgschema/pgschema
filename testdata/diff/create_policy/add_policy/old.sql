CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    tenant_id INTEGER NOT NULL,
    role VARCHAR(50) NOT NULL
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    total NUMERIC(10,2)
);

-- RLS is enabled but no policies exist yet
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
