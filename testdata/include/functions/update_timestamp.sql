--
-- Name: update_timestamp(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS trigger
LANGUAGE plpgsql
STABLE
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;
