CREATE OR REPLACE FUNCTION somefunction(
    param1 text
) RETURNS integer
LANGUAGE sql
AS $$ SELECT length(param1); $$;
