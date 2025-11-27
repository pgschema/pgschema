ALTER FUNCTION calculate_total(numeric, numeric) PARALLEL SAFE;

ALTER FUNCTION calculate_total(numeric, numeric) LEAKPROOF;

ALTER FUNCTION process_data(text) PARALLEL SAFE;

ALTER FUNCTION process_data(text) LEAKPROOF;
