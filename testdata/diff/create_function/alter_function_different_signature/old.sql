CREATE FUNCTION process_order(
    order_id integer,
    discount_percent numeric
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