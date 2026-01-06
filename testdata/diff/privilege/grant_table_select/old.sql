DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'readonly_role') THEN
        CREATE ROLE readonly_role;
    END IF;
END $$;

CREATE TABLE users (id serial PRIMARY KEY);
