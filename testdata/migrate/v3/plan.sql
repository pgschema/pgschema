CREATE TABLE IF NOT EXISTS audit (
    id SERIAL PRIMARY KEY,
    operation text NOT NULL,
    query text,
    user_name text NOT NULL,
    changed_at timestamptz DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_changed_at ON audit (changed_at);

CREATE OR REPLACE FUNCTION log_dml_operations()
RETURNS trigger
LANGUAGE plpgsql
SECURITY INVOKER
VOLATILE
AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES ('INSERT', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = 'UPDATE') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES ('UPDATE', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = 'DELETE') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES ('DELETE', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$;

CREATE OR REPLACE TRIGGER salary_log_trigger
    AFTER UPDATE OR DELETE ON salary
    FOR EACH ROW
    EXECUTE FUNCTION log_dml_operations();
