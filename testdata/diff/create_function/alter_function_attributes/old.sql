CREATE FUNCTION process_data(input text)
RETURNS text
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    RETURN upper(input);
END;
$$;

CREATE FUNCTION calculate_total(amount numeric, tax_rate numeric)
RETURNS numeric
LANGUAGE sql
STABLE
AS $$
    SELECT amount * (1 + tax_rate);
$$;
