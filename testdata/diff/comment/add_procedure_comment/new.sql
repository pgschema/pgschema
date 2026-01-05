CREATE PROCEDURE public.process_order(IN order_id integer)
LANGUAGE sql
AS $$SELECT 1$$;

COMMENT ON PROCEDURE public.process_order(integer) IS 'Processes a single order by ID';
