-- https://github.com/pgschema/pgschema/issues/250
-- Desired state: same default privileges, same table, no explicit grants

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_role') THEN
        CREATE ROLE app_role;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'owner_role') THEN
        CREATE ROLE owner_role;
    END IF;
END $$;

-- Default privileges (same as old state)
ALTER DEFAULT PRIVILEGES FOR ROLE owner_role IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_role;
ALTER DEFAULT PRIVILEGES FOR ROLE owner_role IN SCHEMA public GRANT USAGE ON SEQUENCES TO app_role;

-- Table and sequence (same as old state)
CREATE TABLE users (
    id serial PRIMARY KEY,
    name text NOT NULL
);

-- No explicit grants - covered by default privileges
