--
-- Name: get_order_count; Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION get_order_count(
    user_id_param integer
)
RETURNS integer
LANGUAGE sql
SECURITY INVOKER
VOLATILE
AS $$
    SELECT COUNT(*) FROM orders WHERE user_id = user_id_param;
$$;
