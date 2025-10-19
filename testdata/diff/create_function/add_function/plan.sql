CREATE OR REPLACE FUNCTION process_order(
    order_id integer,
    discount_percent numeric DEFAULT 0,
    priority_level integer DEFAULT 1,
    note varchar DEFAULT '',
    status text DEFAULT 'pending',
    apply_tax boolean DEFAULT true,
    is_priority boolean DEFAULT false
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
