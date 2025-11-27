-- Simple function that returns a default value
CREATE OR REPLACE FUNCTION public.get_default_status()
RETURNS text
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN 'active';
END;
$$;

-- Table with column default that uses the function
CREATE TABLE public.users (
    id serial PRIMARY KEY,
    name text NOT NULL,
    status text DEFAULT get_default_status()
);
