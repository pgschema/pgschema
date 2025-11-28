DROP FUNCTION IF EXISTS process_order(integer, numeric);

CREATE OR REPLACE FUNCTION process_order(
    customer_email text,
    priority boolean
)
RETURNS TABLE(status text, processed_at timestamp)
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
AS $$
BEGIN
    RETURN QUERY
    SELECT 'completed'::text, NOW()
    WHERE priority = true;
END;
$$;
