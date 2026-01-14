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

-- Table-level grant (existing)
GRANT SELECT ON users TO readonly_role;

-- Column-level grant (new - tests column privilege support)
GRANT SELECT (id) ON users TO column_reader;
