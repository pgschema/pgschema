--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: process_order; Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION process_order(
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

