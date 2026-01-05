-- Create roles for testing
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'admin_role') THEN
        CREATE ROLE admin_role;
    END IF;
END $$;

-- Create a table with no explicit grants
CREATE TABLE products (
    id serial PRIMARY KEY,
    name text NOT NULL,
    price numeric(10, 2)
);
