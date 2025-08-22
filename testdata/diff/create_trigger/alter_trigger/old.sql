CREATE TABLE public.employees (
    id serial PRIMARY KEY,
    name text NOT NULL,
    salary numeric(10,2),
    last_modified timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION public.update_last_modified()
RETURNS trigger AS $$
BEGIN
    NEW.last_modified = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER employees_last_modified_trigger
    BEFORE UPDATE ON public.employees
    FOR EACH ROW
    EXECUTE FUNCTION public.update_last_modified();