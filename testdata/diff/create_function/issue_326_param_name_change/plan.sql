DROP FUNCTION IF EXISTS somefunction(text);

CREATE OR REPLACE FUNCTION somefunction(
    new_name text
)
RETURNS text
LANGUAGE sql
VOLATILE
AS $$ SELECT new_name;
$$;
