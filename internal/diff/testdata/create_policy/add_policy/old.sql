CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    tenant_id INTEGER NOT NULL
);

-- RLS is enabled but no policies exist yet
ALTER TABLE users ENABLE ROW LEVEL SECURITY;