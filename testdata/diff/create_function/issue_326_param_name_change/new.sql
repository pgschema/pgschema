CREATE OR REPLACE FUNCTION somefunction(
    new_name text
) RETURNS text
LANGUAGE sql
AS $$ SELECT new_name; $$;
