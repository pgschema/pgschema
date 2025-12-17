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

--
-- Name: extract_domain(text); Type: FUNCTION; Schema: util; Owner: -
--

CREATE FUNCTION util.extract_domain(website text)
  RETURNS text
  LANGUAGE sql
  IMMUTABLE
  PARALLEL SAFE
AS $$
  SELECT CASE WHEN website = ''
    THEN NULL
    ELSE SUBSTRING(website FROM '(?:.*://)?(?:www\.)?([^/?#]*)')
  END
$$;