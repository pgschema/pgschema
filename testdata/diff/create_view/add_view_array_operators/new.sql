CREATE TABLE public.test_data (
    id SERIAL PRIMARY KEY,
    value INTEGER,
    priority INTEGER
);

-- View with various array operators to test ScalarArrayOpExpr formatting
-- The fix ensures that:
-- 1. Only "= ANY" is converted to "IN" syntax
-- 2. Other operators (>, <, <>) preserve "ANY" syntax
CREATE VIEW public.test_array_operators AS
SELECT
    id,
    value,
    -- This SHOULD be converted to IN syntax (= ANY -> IN)
    CASE WHEN value = ANY(ARRAY[10, 20, 30]) THEN 'matched' ELSE 'not_matched' END AS equal_any_test,
    -- These should NOT be converted to IN syntax - they must preserve ANY
    CASE WHEN value > ANY(ARRAY[10, 20, 30]) THEN 'high' ELSE 'low' END AS greater_any_test,
    CASE WHEN value < ANY(ARRAY[5, 15, 25]) THEN 'found_lower' ELSE 'all_higher' END AS less_any_test,
    CASE WHEN priority <> ANY(ARRAY[1, 2, 3]) THEN 'different' ELSE 'same' END AS not_equal_any_test
FROM test_data;
