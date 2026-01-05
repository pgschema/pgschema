CREATE FUNCTION public.calculate_total(price numeric, quantity integer)
RETURNS numeric
LANGUAGE sql
AS $$SELECT price * quantity$$;
