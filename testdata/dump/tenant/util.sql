--
-- Utility functions schema
--

CREATE SCHEMA IF NOT EXISTS util;

--
-- Name: generate_id(); Type: FUNCTION; Schema: util; Owner: -
--

CREATE FUNCTION util.generate_id()
  RETURNS text
  LANGUAGE plpgsql
  STABLE
  PARALLEL SAFE
AS $$
BEGIN
  RETURN 'ID_' || substr(md5(random()::text), 1, 8);
END;
$$;