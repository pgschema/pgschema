CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    tenant_id INTEGER NOT NULL
);

-- Policy has been removed but RLS remains enabled
ALTER TABLE users ENABLE ROW LEVEL SECURITY;