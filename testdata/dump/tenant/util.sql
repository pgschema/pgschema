--
-- Utility functions schema
--

CREATE SCHEMA IF NOT EXISTS utils;

--
-- Name: generate_id(); Type: FUNCTION; Schema: utils; Owner: -
--

CREATE FUNCTION utils.generate_id()
  RETURNS text
  LANGUAGE plpgsql
  STABLE
  PARALLEL SAFE
AS $$
BEGIN
  RETURN 'ID_' || substr(md5(random()::text), 1, 8);
END;
$$;