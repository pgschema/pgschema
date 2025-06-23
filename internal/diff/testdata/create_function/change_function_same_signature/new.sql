CREATE FUNCTION public.process_order(
    order_id integer,
    discount_percent numeric DEFAULT 0
)
RETURNS numeric
LANGUAGE plpgsql
SECURITY INVOKER
STABLE
AS $$
DECLARE
    base_price numeric;
    tax_rate numeric := 0.08;
BEGIN
    -- Different logic: calculate with tax instead of just discount
    SELECT price INTO base_price FROM products WHERE id = order_id;
    RETURN base_price * (1 - discount_percent / 100) * (1 + tax_rate);
END;
$$;