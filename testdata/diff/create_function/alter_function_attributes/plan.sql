ALTER FUNCTION calculate_total(numeric, numeric) PARALLEL SAFE;

ALTER FUNCTION calculate_total(numeric, numeric) LEAKPROOF;

ALTER FUNCTION process_data(text) PARALLEL SAFE;

ALTER FUNCTION process_data(text) LEAKPROOF;

CREATE OR REPLACE FUNCTION secure_lookup(
    id integer
)
RETURNS text
LANGUAGE plpgsql
VOLATILE
SET search_path = pg_catalog
AS $$
BEGIN
    RETURN 'result';
END;
$$;
