CREATE OR REPLACE FUNCTION somefunction(
    old_name text
) RETURNS text
LANGUAGE sql
AS $$ SELECT old_name; $$;
