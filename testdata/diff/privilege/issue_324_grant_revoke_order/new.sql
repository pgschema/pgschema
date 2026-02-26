DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
        CREATE ROLE app_user;
    END IF;
END $$;

CREATE TABLE sometable (somecolumn text);

GRANT UPDATE (somecolumn) ON sometable TO app_user;
