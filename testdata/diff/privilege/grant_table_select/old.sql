DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'readonly_role') THEN
        CREATE ROLE readonly_role;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'column_reader') THEN
        CREATE ROLE column_reader;
    END IF;
END $$;

CREATE TABLE users (id serial PRIMARY KEY);
