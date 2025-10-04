CREATE OR REPLACE TRIGGER employees_insert_timestamp_trigger
    AFTER INSERT ON employees
    FOR EACH ROW
    EXECUTE FUNCTION update_last_modified();

CREATE OR REPLACE TRIGGER employees_last_modified_trigger
    BEFORE UPDATE ON employees
    FOR EACH ROW
    EXECUTE FUNCTION update_last_modified();

CREATE OR REPLACE TRIGGER employees_truncate_log_trigger
    AFTER TRUNCATE ON employees
    FOR EACH STATEMENT
    EXECUTE FUNCTION update_last_modified();
