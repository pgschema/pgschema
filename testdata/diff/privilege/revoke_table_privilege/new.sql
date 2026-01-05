-- Create roles for testing
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'old_role') THEN
        CREATE ROLE old_role;
    END IF;
END $$;

-- Create a table with no grants (privilege revoked)
CREATE TABLE audit_log (
    id serial PRIMARY KEY,
    action text NOT NULL,
    created_at timestamp DEFAULT now()
);
