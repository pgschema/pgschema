CREATE FUNCTION process_order(
    order_id integer,
    discount_percent numeric DEFAULT 0
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

-- Function with OUT parameters - tests that DROP only includes IN parameters
CREATE FUNCTION get_user_stats(
    user_id integer,
    OUT total_orders integer,
    OUT total_amount numeric,
    OUT last_order_date timestamp
)
LANGUAGE plpgsql
SECURITY INVOKER
STABLE
AS $$
BEGIN
    SELECT
        COUNT(*),
        COALESCE(SUM(amount), 0),
        MAX(order_date)
    INTO total_orders, total_amount, last_order_date
    FROM orders
    WHERE orders.user_id = get_user_stats.user_id;
END;
$$;

-- Function with mixed IN, OUT, and INOUT parameters
CREATE FUNCTION process_payment(
    IN order_id integer,
    INOUT transaction_id text,
    OUT status text,
    OUT processed_at timestamp
)
LANGUAGE plpgsql
SECURITY INVOKER
VOLATILE
AS $$
BEGIN
    -- Process payment logic
    transaction_id := 'TXN-' || transaction_id;
    status := 'SUCCESS';
    processed_at := NOW();
END;
$$;

-- Role and grant for testing REVOKE ordering
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'api_role') THEN
        CREATE ROLE api_role;
    END IF;
END $$;

GRANT EXECUTE ON FUNCTION process_order(integer, numeric) TO api_role;