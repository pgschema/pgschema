-- Create roles for testing
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'api_role') THEN
        CREATE ROLE api_role;
    END IF;
END $$;

-- Create a function with default PUBLIC execute
CREATE FUNCTION calculate_total(quantity integer, unit_price numeric)
RETURNS numeric
LANGUAGE sql
AS $$
    SELECT quantity * unit_price;
$$;
