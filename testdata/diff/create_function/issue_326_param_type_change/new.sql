CREATE OR REPLACE FUNCTION somefunction(
    param2 uuid
) RETURNS uuid
LANGUAGE sql
AS $$ SELECT param2; $$;
