-- Create roles for testing
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'old_role') THEN
        CREATE ROLE old_role;
    END IF;
END $$;

-- Create a table
CREATE TABLE audit_log (
    id serial PRIMARY KEY,
    action text NOT NULL,
    created_at timestamp DEFAULT now()
);

-- Grant SELECT to old_role
GRANT SELECT ON audit_log TO old_role;
