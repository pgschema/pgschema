DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'api_role') THEN
        CREATE ROLE api_role;
    END IF;
END $$;

CREATE FUNCTION calculate_total(quantity integer, unit_price numeric)
RETURNS numeric
LANGUAGE sql
AS $$ SELECT quantity * unit_price; $$;

GRANT EXECUTE ON FUNCTION calculate_total(integer, numeric) TO api_role;
