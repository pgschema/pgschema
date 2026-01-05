-- Create roles for testing
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'admin_user') THEN
        CREATE ROLE admin_user;
    END IF;
END $$;

-- Grant with grant option - admin_user can grant to others
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT ON TABLES TO admin_user WITH GRANT OPTION;

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);
