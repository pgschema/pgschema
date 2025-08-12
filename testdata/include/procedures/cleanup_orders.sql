--
-- Name: cleanup_orders; Type: PROCEDURE; Schema: -; Owner: -
--

CREATE OR REPLACE PROCEDURE cleanup_orders()
LANGUAGE sql
AS $$
    DELETE FROM orders WHERE status = 'completed';
$$;
