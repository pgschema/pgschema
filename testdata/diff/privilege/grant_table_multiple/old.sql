-- Create roles for testing
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_role') THEN
        CREATE ROLE app_role;
    END IF;
END $$;

-- Create a table with no explicit grants
CREATE TABLE orders (
    id serial PRIMARY KEY,
    user_id integer NOT NULL,
    total numeric(10, 2)
);
