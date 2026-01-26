-- Create role for testing
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
        CREATE ROLE app_user;
    END IF;
END $$;

-- Default privileges grant SELECT, INSERT, UPDATE, DELETE on all new tables
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_user;

-- Create a read-only table - user should only have SELECT, not INSERT/UPDATE/DELETE
CREATE TABLE readonly_data (
    id integer PRIMARY KEY,
    value text
);

-- Revoke write privileges - keep only SELECT from the default grants
REVOKE INSERT, UPDATE, DELETE ON TABLE readonly_data FROM app_user;
