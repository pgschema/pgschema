--
-- Test case for GitHub issue #125: Function parameter defaults being lost
--
-- This test case reproduces a bug where function parameter default values
-- are lost when dumping functions from the database.
--
-- The issue manifests when:
-- 1. A function has parameters with DEFAULT values
-- 2. The function is dumped using pgschema
-- 3. The dumped output is missing the DEFAULT clauses
--
-- Examples from the bug report:
-- - raise_on_error BOOLEAN DEFAULT TRUE becomes raise_on_error boolean
-- - p_schema_override text DEFAULT NULL::text becomes p_schema_override text DEFAULT NULL (sometimes preserved)
--

--
-- Test case 1: Simple function with only IN parameters and defaults
-- This tests the basic case where defaults should be preserved
--
CREATE OR REPLACE FUNCTION test_simple_defaults(
    param1 integer DEFAULT 42,
    param2 text DEFAULT 'hello world',
    param3 boolean DEFAULT true,
    param4 numeric DEFAULT 3.14,
    param5 timestamp DEFAULT CURRENT_TIMESTAMP
)
RETURNS text
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN 'Params: ' || param1 || ', ' || param2 || ', ' || param3 || ', ' || param4 || ', ' || param5;
END;
$$;

--
-- Test case 2: Function with mixed IN/OUT parameters and defaults
-- This tests the case where parseParametersFromProcArrays is used
--
CREATE OR REPLACE FUNCTION test_mixed_params(
    IN raise_on_error BOOLEAN DEFAULT TRUE,
    IN p_schema_override text DEFAULT NULL::text,
    IN max_retries integer DEFAULT 3,
    OUT success boolean,
    OUT message text
)
LANGUAGE plpgsql
AS $$
BEGIN
    success := NOT raise_on_error;
    message := 'Schema: ' || COALESCE(p_schema_override, 'default') || ', Retries: ' || max_retries;
END;
$$;

--
-- Test case 3: Function with INOUT parameters and defaults
-- This also triggers the array-based parsing path
--
CREATE OR REPLACE FUNCTION test_inout_params(
    value integer DEFAULT 100,
    INOUT multiplier numeric DEFAULT 1.5,
    INOUT result numeric DEFAULT NULL
)
LANGUAGE plpgsql
AS $$
BEGIN
    result := value * multiplier;
    multiplier := multiplier * 2;
END;
$$;

--
-- Test case 4: Function with complex default expressions
-- Tests preservation of complex default values
--
CREATE OR REPLACE FUNCTION test_complex_defaults(
    arr integer[] DEFAULT ARRAY[1, 2, 3],
    json_data jsonb DEFAULT '{"key": "value"}'::jsonb,
    range_val int4range DEFAULT '[1,10)'::int4range,
    expr_default integer DEFAULT (2 + 2) * 10
)
RETURNS jsonb
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN jsonb_build_object(
        'arr', arr,
        'json', json_data,
        'range', range_val::text,
        'expr', expr_default
    );
END;
$$;

--
-- Test case 5: Function with VARIADIC parameter and defaults
-- Tests edge case with VARIADIC
--
CREATE OR REPLACE FUNCTION test_variadic_defaults(
    prefix text DEFAULT 'Result:',
    VARIADIC numbers integer[] DEFAULT ARRAY[]::integer[]
)
RETURNS text
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN prefix || ' ' || array_to_string(numbers, ', ');
END;
$$;