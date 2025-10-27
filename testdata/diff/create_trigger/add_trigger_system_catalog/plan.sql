CREATE OR REPLACE TRIGGER employees_update_check
    BEFORE UPDATE ON employees
    FOR EACH ROW
    EXECUTE FUNCTION pg_catalog.suppress_redundant_updates_trigger();
