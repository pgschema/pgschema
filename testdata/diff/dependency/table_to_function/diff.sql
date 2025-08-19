CREATE TABLE IF NOT EXISTS documents (
    id SERIAL PRIMARY KEY,
    title text NOT NULL,
    content text,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION get_document_count()
RETURNS integer
LANGUAGE plpgsql
SECURITY INVOKER
VOLATILE
AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM public.documents);
END;
$$;
