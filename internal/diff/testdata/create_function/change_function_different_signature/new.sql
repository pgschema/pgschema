CREATE FUNCTION process_order(
    customer_email text,
    priority boolean DEFAULT false
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