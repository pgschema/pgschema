DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_role') THEN
        CREATE ROLE app_role;
    END IF;
END $$;

CREATE SEQUENCE order_id_seq;

GRANT USAGE, SELECT ON SEQUENCE order_id_seq TO app_role;
