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

CREATE TRIGGER products_description_trigger
    BEFORE UPDATE ON public.products
    FOR EACH ROW
    WHEN (NEW.description IS DISTINCT FROM OLD.description)
    EXECUTE FUNCTION public.log_description_change();

CREATE TRIGGER products_status_trigger
    BEFORE UPDATE ON public.products
    FOR EACH ROW
    WHEN (NEW.status IS NOT DISTINCT FROM OLD.status)
    EXECUTE FUNCTION public.skip_status_change();