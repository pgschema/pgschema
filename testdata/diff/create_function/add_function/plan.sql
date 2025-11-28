CREATE OR REPLACE FUNCTION calculate_tax(
    amount numeric,
    rate numeric
)
RETURNS numeric
LANGUAGE sql
IMMUTABLE
PARALLEL SAFE
AS $$
    SELECT amount * rate;
$$;

CREATE OR REPLACE FUNCTION mask_sensitive_data(
    input text
)
RETURNS text
LANGUAGE sql
STABLE
LEAKPROOF
AS $$
    SELECT '***' || substring(input from 4);
$$;

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
VOLATILE
STRICT
SECURITY DEFINER
LEAKPROOF
PARALLEL RESTRICTED
AS $$
DECLARE
    total numeric;
BEGIN
    SELECT amount INTO total FROM orders WHERE id = order_id;
    RETURN total - (total * discount_percent / 100);
END;
$$;
