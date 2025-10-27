--
-- PostgreSQL database dump
--

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: test_simple_defaults(integer, text, boolean, numeric, timestamp without time zone); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.test_simple_defaults(param1 integer DEFAULT 42, param2 text DEFAULT 'hello world'::text, param3 boolean DEFAULT true, param4 numeric DEFAULT 3.14, param5 timestamp without time zone DEFAULT CURRENT_TIMESTAMP) RETURNS text
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN 'Params: ' || param1 || ', ' || param2 || ', ' || param3 || ', ' || param4 || ', ' || param5;
END;
$$;

--
-- Name: test_mixed_params(boolean, text, integer); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.test_mixed_params(raise_on_error boolean DEFAULT true, p_schema_override text DEFAULT NULL::text, max_retries integer DEFAULT 3, OUT success boolean, OUT message text) RETURNS record
    LANGUAGE plpgsql
    AS $$
BEGIN
    success := NOT raise_on_error;
    message := 'Schema: ' || COALESCE(p_schema_override, 'default') || ', Retries: ' || max_retries;
END;
$$;

--
-- Name: test_inout_params(integer, numeric, numeric); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.test_inout_params(value integer DEFAULT 100, INOUT multiplier numeric DEFAULT 1.5, INOUT result numeric DEFAULT NULL::numeric) RETURNS record
    LANGUAGE plpgsql
    AS $$
BEGIN
    result := value * multiplier;
    multiplier := multiplier * 2;
END;
$$;

--
-- Name: test_complex_defaults(integer[], jsonb, int4range, integer); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.test_complex_defaults(arr integer[] DEFAULT ARRAY[1, 2, 3], json_data jsonb DEFAULT '{"key": "value"}'::jsonb, range_val int4range DEFAULT '[1,10)'::int4range, expr_default integer DEFAULT 40) RETURNS jsonb
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
-- Name: test_variadic_defaults(text, integer[]); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.test_variadic_defaults(prefix text DEFAULT 'Result:'::text, VARIADIC numbers integer[] DEFAULT ARRAY[]::integer[]) RETURNS text
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN prefix || ' ' || array_to_string(numbers, ', ');
END;
$$;

--
-- PostgreSQL database dump complete
--