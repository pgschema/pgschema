CREATE OR REPLACE FUNCTION z_helper(
    input text
)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$ SELECT upper(input)
$$;
CREATE OR REPLACE FUNCTION a_wrapper(
    input text
)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$ SELECT z_helper(input)
$$;
