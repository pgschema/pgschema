CREATE FUNCTION public.calculate_total(price numeric, quantity integer)
RETURNS numeric
LANGUAGE sql
AS $$SELECT price * quantity$$;

COMMENT ON FUNCTION public.calculate_total(numeric, integer) IS 'Calculates total price from unit price and quantity';
