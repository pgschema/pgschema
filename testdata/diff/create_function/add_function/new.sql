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
    is_priority boolean DEFAULT false
)
RETURNS numeric
LANGUAGE plpgsql
VOLATILE
PARALLEL RESTRICTED
LEAKPROOF
SECURITY DEFINER
STRICT
AS $$
DECLARE
    total numeric;
BEGIN
    SELECT amount INTO total FROM orders WHERE id = order_id;
    RETURN total - (total * discount_percent / 100);
END;
$$;

-- Table function with RETURN clause (bug report test case)
CREATE FUNCTION days_since_special_date() RETURNS SETOF timestamptz
    LANGUAGE sql
    STABLE
    PARALLEL SAFE
    LEAKPROOF
    RETURN generate_series(date_trunc('day', '2025-01-01'::timestamp), date_trunc('day', NOW()), '1 day'::interval);

-- Simple pure function demonstrating PARALLEL SAFE + LEAKPROOF
CREATE FUNCTION safe_add(a integer, b integer)
RETURNS integer
LANGUAGE sql
IMMUTABLE
PARALLEL SAFE
LEAKPROOF
STRICT
AS $$
    SELECT a + b;
$$;