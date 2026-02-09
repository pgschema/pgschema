-- https://github.com/pgplex/pgschema/issues/250
--
-- Test: Privileges covered by default privileges should not be revoked.
--
-- This test simulates the scenario where:
-- 1. Default privileges are configured for owner_role
-- 2. Objects were created by owner_role, so privileges were auto-granted
-- 3. The explicit GRANTs below represent those auto-granted privileges
--
-- In a real database, these grants would be created automatically by PostgreSQL
-- when owner_role creates objects. Here we simulate this by adding explicit GRANTs.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_role') THEN
        CREATE ROLE app_role;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'owner_role') THEN
        CREATE ROLE owner_role;
    END IF;
END $$;

-- Default privileges for owner_role
ALTER DEFAULT PRIVILEGES FOR ROLE owner_role IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_role;
ALTER DEFAULT PRIVILEGES FOR ROLE owner_role IN SCHEMA public GRANT USAGE ON SEQUENCES TO app_role;

-- Table and sequence
CREATE TABLE users (
    id serial PRIMARY KEY,
    name text NOT NULL
);

-- Simulate auto-granted privileges (what PostgreSQL would grant automatically
-- when owner_role creates objects with the above default privileges)
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE users TO app_role;
GRANT USAGE ON SEQUENCE users_id_seq TO app_role;
