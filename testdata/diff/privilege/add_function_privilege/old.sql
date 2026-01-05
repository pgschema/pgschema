-- Create roles for testing
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'api_user') THEN
        CREATE ROLE api_user;
    END IF;
END $$;

-- No default privileges configured
CREATE FUNCTION get_version() RETURNS text AS $$
BEGIN
    RETURN '1.0.0';
END;
$$ LANGUAGE plpgsql;
