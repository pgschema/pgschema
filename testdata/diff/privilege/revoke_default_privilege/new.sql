-- Create role for testing
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'reader') THEN
        CREATE ROLE reader;
    END IF;
END $$;

-- Default privileges grant SELECT on all new tables to reader
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO reader;

-- Create a table that should NOT inherit the default SELECT privilege
CREATE TABLE secrets (
    id integer PRIMARY KEY,
    data text
);

-- Explicitly revoke the auto-granted default privilege
REVOKE SELECT ON TABLE secrets FROM reader;
