--
-- Test case for GitHub issue #191: Overloaded functions and procedures not fully dumped
--
-- This test case reproduces a bug where only the last overloaded function/procedure
-- is included in the dump output. Functions and procedures are stored by name only,
-- causing overloads with different signatures to overwrite each other.
--

--
-- Function overloads: 3 versions of test_func with different signatures
--

-- Overload 1: Single integer parameter
CREATE OR REPLACE FUNCTION test_func(a integer)
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN a * 2;
END;
$$;

-- Overload 2: Two integer parameters (different count)
CREATE OR REPLACE FUNCTION test_func(a integer, b integer)
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN a + b;
END;
$$;

-- Overload 3: Single text parameter (different type)
CREATE OR REPLACE FUNCTION test_func(a text)
RETURNS text
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN 'Hello, ' || a;
END;
$$;

--
-- Procedure overloads: 3 versions of test_proc with different signatures
--

-- Overload 1: Single integer parameter
CREATE OR REPLACE PROCEDURE test_proc(a integer)
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE NOTICE 'Integer: %', a;
END;
$$;

-- Overload 2: Two integer parameters (different count)
CREATE OR REPLACE PROCEDURE test_proc(a integer, b integer)
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE NOTICE 'Sum: %', a + b;
END;
$$;

-- Overload 3: Single text parameter (different type)
CREATE OR REPLACE PROCEDURE test_proc(a text)
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE NOTICE 'Text: %', a;
END;
$$;
