CREATE TABLE public.employees (
    id serial PRIMARY KEY,
    name text NOT NULL,
    salary numeric(10,2),
    last_modified timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION public.update_last_modified()
RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'TRUNCATE' THEN
        RAISE NOTICE 'Table truncated';
        RETURN NULL;
    END IF;
    NEW.last_modified = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER employees_last_modified_trigger
    BEFORE UPDATE ON public.employees
    FOR EACH ROW
    EXECUTE FUNCTION public.update_last_modified();

CREATE TRIGGER employees_insert_timestamp_trigger
    AFTER INSERT ON public.employees
    FOR EACH ROW
    EXECUTE FUNCTION public.update_last_modified();

CREATE TRIGGER employees_truncate_log_trigger
    AFTER TRUNCATE ON public.employees
    FOR EACH STATEMENT
    EXECUTE FUNCTION public.update_last_modified();