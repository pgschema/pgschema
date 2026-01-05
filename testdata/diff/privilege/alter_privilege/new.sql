-- Create roles for testing
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_role') THEN
        CREATE ROLE app_role;
    END IF;
END $$;

-- Create a table
CREATE TABLE inventory (
    id serial PRIMARY KEY,
    product_name text NOT NULL,
    quantity integer DEFAULT 0
);

-- Change privileges: remove INSERT, add UPDATE and DELETE
GRANT SELECT, UPDATE, DELETE ON inventory TO app_role;
