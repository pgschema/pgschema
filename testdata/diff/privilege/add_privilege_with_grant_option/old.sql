-- Create roles for testing
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'admin_user') THEN
        CREATE ROLE admin_user;
    END IF;
END $$;

-- No default privileges configured
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);
