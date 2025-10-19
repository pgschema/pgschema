DROP FUNCTION IF EXISTS process_order(integer, numeric);

CREATE OR REPLACE FUNCTION process_order(
    customer_email text,
    priority boolean
)
RETURNS TABLE(status text, processed_at timestamp)
LANGUAGE plpgsql
SECURITY DEFINER
STABLE
AS $$
BEGIN
    RETURN QUERY
    SELECT 'completed'::text, NOW()
    WHERE priority = true;
END;
$$;
