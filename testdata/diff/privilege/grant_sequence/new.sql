-- Create roles for testing
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_role') THEN
        CREATE ROLE app_role;
    END IF;
END $$;

-- Create a sequence
CREATE SEQUENCE order_id_seq;

-- Grant USAGE and SELECT on sequence to app_role
GRANT USAGE, SELECT ON SEQUENCE order_id_seq TO app_role;
