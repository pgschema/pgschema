--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: documents; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS documents (
    id SERIAL PRIMARY KEY,
    title text NOT NULL,
    content text,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP
);

--
-- Name: get_document_count; Type: FUNCTION; Schema: -; Owner: -
--

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

