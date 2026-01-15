CREATE PROCEDURE example_procedure(
    IN input_value integer,
    OUT output_value integer
)
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE NOTICE 'Input value is: %', input_value;
    output_value := input_value + 1;
END;
$$;

-- Procedure testing BEGIN ATOMIC syntax (SQL-standard body, PG14+)
-- Reproduces issue #241 for procedures
CREATE PROCEDURE validate_input(input_value integer)
LANGUAGE SQL
BEGIN ATOMIC
    SELECT input_value * 2;
END;