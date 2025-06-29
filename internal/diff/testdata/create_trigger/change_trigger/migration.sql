CREATE OR REPLACE TRIGGER employees_update_check
    BEFORE INSERT OR UPDATE ON public.employees
    FOR EACH ROW
    WHEN (NEW.salary IS NOT NULL)
    EXECUTE FUNCTION pg_catalog.suppress_redundant_updates_trigger();