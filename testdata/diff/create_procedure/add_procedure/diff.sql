CREATE OR REPLACE PROCEDURE example_procedure(
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
