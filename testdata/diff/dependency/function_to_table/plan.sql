CREATE OR REPLACE FUNCTION get_default_status()
RETURNS text
LANGUAGE plpgsql
SECURITY INVOKER
VOLATILE
AS $$
BEGIN
    RETURN 'active';
END;
$$;

CREATE TABLE IF NOT EXISTS users (
    id SERIAL,
    name text NOT NULL,
    status text DEFAULT get_default_status(),
    CONSTRAINT users_pkey PRIMARY KEY (id)
);
