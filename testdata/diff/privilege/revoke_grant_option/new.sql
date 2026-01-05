DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'manager_role') THEN
        CREATE ROLE manager_role;
    END IF;
END $$;

CREATE TABLE employees (id serial PRIMARY KEY);

GRANT SELECT ON employees TO manager_role;
