-- Create roles for testing
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'api_role') THEN
        CREATE ROLE api_role;
    END IF;
END $$;

-- Create a function
CREATE FUNCTION calculate_total(quantity integer, unit_price numeric)
RETURNS numeric
LANGUAGE sql
AS $$
    SELECT quantity * unit_price;
$$;

-- Grant EXECUTE to api_role
GRANT EXECUTE ON FUNCTION calculate_total(integer, numeric) TO api_role;
