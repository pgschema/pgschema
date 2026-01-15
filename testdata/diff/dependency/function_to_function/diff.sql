CREATE OR REPLACE FUNCTION get_raw_result()
RETURNS integer
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    RETURN 42;
END;
$$;

CREATE OR REPLACE FUNCTION get_formatted_result()
RETURNS text
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    RETURN 'Result: ' || get_raw_result()::text;
END;
$$;
