CREATE OR REPLACE TRIGGER employees_update_check
    BEFORE UPDATE ON employees
    FOR EACH ROW
    EXECUTE FUNCTION suppress_redundant_updates_trigger();
