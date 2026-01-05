-- Create roles for testing
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
        CREATE ROLE app_user;
    END IF;
END $$;

-- Grant USAGE on future types to app_user
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE ON TYPES TO app_user;

CREATE TYPE status AS ENUM ('pending', 'active', 'inactive');
