-- z_helper must be created first because a_wrapper calls it
-- Without body dependency scanning, alphabetical order would create a_wrapper first and fail

CREATE FUNCTION z_helper(input text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$ SELECT upper(input) $$;

CREATE FUNCTION a_wrapper(input text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$ SELECT z_helper(input) $$;
