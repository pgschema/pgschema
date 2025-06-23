CREATE OR REPLACE FUNCTION public.process_order(
    order_id integer,
    discount_percent numeric DEFAULT 0
)
RETURNS numeric
LANGUAGE plpgsql
SECURITY DEFINER
VOLATILE
AS $$
DECLARE
    total numeric;
BEGIN
    SELECT amount INTO total FROM orders WHERE id = order_id;
    RETURN total - (total * discount_percent / 100);
END;
$$;