-- https://github.com/pgschema/pgschema/issues/250
-- Current state: default privileges + explicit grants that match them

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_role') THEN
        CREATE ROLE app_role;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'owner_role') THEN
        CREATE ROLE owner_role;
    END IF;
END $$;

-- Default privileges
ALTER DEFAULT PRIVILEGES FOR ROLE owner_role IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_role;
ALTER DEFAULT PRIVILEGES FOR ROLE owner_role IN SCHEMA public GRANT USAGE ON SEQUENCES TO app_role;

-- Table and sequence
CREATE TABLE users (
    id serial PRIMARY KEY,
    name text NOT NULL
);

-- Explicit grants that match the default privileges above
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE users TO app_role;
GRANT USAGE ON SEQUENCE users_id_seq TO app_role;
