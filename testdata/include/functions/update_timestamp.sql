--
-- Name: update_timestamp; Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS trigger
LANGUAGE plpgsql
SECURITY INVOKER
STABLE
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;