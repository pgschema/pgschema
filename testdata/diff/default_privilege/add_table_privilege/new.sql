-- Create roles for testing
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
        CREATE ROLE app_user;
    END IF;
END $$;

-- Grant SELECT on future tables to PUBLIC
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO PUBLIC;

-- Grant INSERT, UPDATE to app_user role
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT INSERT, UPDATE ON TABLES TO app_user;

-- Create a new table - default privileges should apply automatically
CREATE TABLE users (
    id integer PRIMARY KEY,
    name text NOT NULL
);
