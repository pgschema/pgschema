CREATE FUNCTION process_order(
    order_id integer,
    discount_percent numeric DEFAULT 0,
    note varchar DEFAULT ''
)
RETURNS numeric
LANGUAGE plpgsql
SECURITY DEFINER
VOLATILE
STRICT
AS $$
DECLARE
    total numeric;
BEGIN
    SELECT amount INTO total FROM orders WHERE id = order_id;
    RETURN total - (total * discount_percent / 100);
END;
$$;