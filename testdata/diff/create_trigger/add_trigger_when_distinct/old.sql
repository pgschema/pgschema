CREATE TABLE public.products (
    id serial PRIMARY KEY,
    name text NOT NULL,
    description text,
    status text,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION public.log_description_change()
RETURNS trigger AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION public.skip_status_change()
RETURNS trigger AS $$
BEGIN
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;