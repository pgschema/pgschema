-- No default privileges configured
CREATE FUNCTION get_version() RETURNS text AS $$
BEGIN
    RETURN '1.0.0';
END;
$$ LANGUAGE plpgsql;
