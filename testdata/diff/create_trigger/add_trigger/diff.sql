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

CREATE OR REPLACE FUNCTION update_last_modified()
RETURNS trigger
LANGUAGE plpgsql
SECURITY INVOKER
VOLATILE
AS $$
BEGIN
    IF TG_OP = 'TRUNCATE' THEN
        RAISE NOTICE 'Table truncated';
        RETURN NULL;
    END IF;
    NEW.last_modified = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;
