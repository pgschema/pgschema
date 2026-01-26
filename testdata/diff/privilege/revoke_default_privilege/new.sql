-- Create roles for testing
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'reader') THEN
        CREATE ROLE reader;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
        CREATE ROLE app_user;
    END IF;
END $$;

-- Default privileges grant SELECT on all new tables to reader
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO reader;

-- Default privileges grant SELECT, INSERT, UPDATE, DELETE on all new tables to app_user
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_user;

-- Create a table that should NOT inherit any default privileges for reader (full revoke)
CREATE TABLE secrets (
    id integer PRIMARY KEY,
    data text
);

-- Explicitly revoke the auto-granted SELECT from reader
REVOKE SELECT ON TABLE secrets FROM reader;

-- Create a read-only table - app_user should only have SELECT, not INSERT/UPDATE/DELETE (partial revoke)
CREATE TABLE readonly_data (
    id integer PRIMARY KEY,
    value text
);

-- Revoke write privileges from app_user - keep only SELECT from the default grants
REVOKE INSERT, UPDATE, DELETE ON TABLE readonly_data FROM app_user;
