CREATE OR REPLACE FUNCTION days_since_special_date()
RETURNS SETOF timestamp with time zone
LANGUAGE sql
STABLE
PARALLEL SAFE
RETURN generate_series((date_trunc('day'::text, '2025-01-01 00:00:00'::timestamp without time zone))::timestamp with time zone, date_trunc('day'::text, now()), '1 day'::interval);

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
AS $$
DECLARE
    total numeric;
BEGIN
    SELECT amount INTO total FROM orders WHERE id = order_id;
    RETURN total - (total * discount_percent / 100);
END;
$$;
