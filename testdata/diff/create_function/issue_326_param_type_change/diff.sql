DROP FUNCTION IF EXISTS somefunction(text);

CREATE OR REPLACE FUNCTION somefunction(
    param2 uuid
)
RETURNS uuid
LANGUAGE sql
VOLATILE
AS $$ SELECT param2;
$$;
