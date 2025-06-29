CREATE TABLE public.employees (
    id serial PRIMARY KEY,
    name text NOT NULL,
    salary numeric(10,2),
    last_modified timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER employees_update_check
    BEFORE INSERT OR UPDATE ON public.employees
    FOR EACH ROW
    WHEN (NEW.salary IS NOT NULL)
    EXECUTE FUNCTION pg_catalog.suppress_redundant_updates_trigger();