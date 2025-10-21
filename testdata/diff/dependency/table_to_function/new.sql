CREATE TABLE public.documents (
    id serial PRIMARY KEY,
    title text NOT NULL,
    content text,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION public.get_document_count()
RETURNS integer AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM public.documents);
END;
$$ LANGUAGE plpgsql;
