-- https://github.com/pgplex/pgschema/issues/250
--
-- Test: Privileges covered by default privileges should not be revoked.
--
-- This represents the desired state as written in the user's SQL files.
-- The user declares default privileges and creates objects, but does NOT
-- include explicit GRANTs because they expect PostgreSQL to auto-grant them.
--
-- The diff should NOT generate REVOKE statements because the privileges
-- in old.sql are covered by the default privileges defined here.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_role') THEN
        CREATE ROLE app_role;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'owner_role') THEN
        CREATE ROLE owner_role;
    END IF;
END $$;

-- Default privileges for owner_role (same as old state)
ALTER DEFAULT PRIVILEGES FOR ROLE owner_role IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_role;
ALTER DEFAULT PRIVILEGES FOR ROLE owner_role IN SCHEMA public GRANT USAGE ON SEQUENCES TO app_role;

-- Table and sequence (same as old state)
CREATE TABLE users (
    id serial PRIMARY KEY,
    name text NOT NULL
);

-- No explicit GRANTs - covered by default privileges above
