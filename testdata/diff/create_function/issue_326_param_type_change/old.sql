CREATE OR REPLACE FUNCTION somefunction(
    param1 text
) RETURNS text
LANGUAGE sql
AS $$ SELECT param1; $$;
