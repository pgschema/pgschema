DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'old_role') THEN
        CREATE ROLE old_role;
    END IF;
END $$;

CREATE TABLE audit_log (id serial PRIMARY KEY);
