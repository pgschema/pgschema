--
-- Name: get_user_count; Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION get_user_count()
RETURNS integer
LANGUAGE sql
SECURITY INVOKER
VOLATILE
AS $$
    SELECT COUNT(*) FROM users;
$$;
