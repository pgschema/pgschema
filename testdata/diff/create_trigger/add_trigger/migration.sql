CREATE OR REPLACE TRIGGER employees_last_modified_trigger
    BEFORE UPDATE ON employees
    FOR EACH ROW
    EXECUTE FUNCTION update_last_modified();