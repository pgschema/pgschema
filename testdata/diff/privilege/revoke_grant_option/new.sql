-- Create roles for testing
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'manager_role') THEN
        CREATE ROLE manager_role;
    END IF;
END $$;

-- Create a table
CREATE TABLE employees (
    id serial PRIMARY KEY,
    name text NOT NULL,
    department text
);

-- Grant SELECT without grant option (grant option removed)
GRANT SELECT ON employees TO manager_role;
