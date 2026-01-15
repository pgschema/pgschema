-- Base function (dependency)
CREATE OR REPLACE FUNCTION public.get_raw_result()
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN 42;
END;
$$;

-- Dependent function that calls the base function
CREATE OR REPLACE FUNCTION public.get_formatted_result()
RETURNS text
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN 'Result: ' || get_raw_result()::text;
END;
$$;
