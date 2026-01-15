-- Complex function demonstrating all qualifiers
CREATE FUNCTION process_order(
    order_id integer,
    -- Simple numeric defaults
    discount_percent numeric DEFAULT 0,
    priority_level integer DEFAULT 1,
    -- String defaults
    note varchar DEFAULT '',
    status text DEFAULT 'pending',
    -- Boolean defaults
    apply_tax boolean DEFAULT true,
    is_priority boolean DEFAULT false,
    -- Interval default (reproduces issue #216)
    expiry_date date DEFAULT (CURRENT_DATE + INTERVAL '1 year')
)
RETURNS numeric
LANGUAGE plpgsql
VOLATILE
STRICT
SECURITY DEFINER
LEAKPROOF
PARALLEL RESTRICTED
SET search_path = pg_catalog, public
AS $$
DECLARE
    total numeric;
BEGIN
    SELECT amount INTO total FROM orders WHERE id = order_id;
    RETURN total - (total * discount_percent / 100);
END;
$$;

-- Function testing PARALLEL SAFE only
CREATE FUNCTION calculate_tax(amount numeric, rate numeric)
RETURNS numeric
LANGUAGE sql
IMMUTABLE
PARALLEL SAFE
AS $$
    SELECT amount * rate;
$$;

-- Function testing LEAKPROOF only
CREATE FUNCTION mask_sensitive_data(input text)
RETURNS text
LANGUAGE sql
STABLE
LEAKPROOF
AS $$
    SELECT '***' || substring(input from 4);
$$;

-- Function testing BEGIN ATOMIC syntax (SQL-standard multi-statement body, PG14+)
-- Reproduces issue #241
CREATE FUNCTION add_with_tax(amount numeric, tax_rate numeric DEFAULT 0.1)
RETURNS numeric
LANGUAGE SQL
BEGIN ATOMIC
    SELECT amount + (amount * tax_rate);
END;
