CREATE OR REPLACE VIEW test_array_operators AS
 SELECT
    id,
    value,
    CASE WHEN value IN (10, 20, 30) THEN 'matched' ELSE 'not_matched' END AS equal_any_test,
    CASE WHEN value > ANY (ARRAY[10, 20, 30]) THEN 'high' ELSE 'low' END AS greater_any_test,
    CASE WHEN value < ANY (ARRAY[5, 15, 25]) THEN 'found_lower' ELSE 'all_higher' END AS less_any_test,
    CASE WHEN priority <> ANY (ARRAY[1, 2, 3]) THEN 'different' ELSE 'same' END AS not_equal_any_test
   FROM test_data;
