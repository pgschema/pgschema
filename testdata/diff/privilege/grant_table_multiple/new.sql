DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_role') THEN
        CREATE ROLE app_role;
    END IF;
END $$;

CREATE TABLE orders (id serial PRIMARY KEY);

GRANT SELECT, INSERT, UPDATE, DELETE ON orders TO app_role;
