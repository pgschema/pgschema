--
-- Test case for GitHub issue #320: Reserved keywords after schema prefix stripping
--
-- When a function body contains schema-qualified references like public.user,
-- stripping the schema prefix should produce "user" (quoted) since user is
-- a reserved keyword. Without quoting, the dumped SQL is syntactically invalid.
--

CREATE TABLE "user" (
    id serial PRIMARY KEY,
    name text NOT NULL,
    email text
);

-- Function using schema-qualified reserved keyword type in DECLARE and body
CREATE OR REPLACE FUNCTION get_first_user()
RETURNS text
LANGUAGE plpgsql
AS $$
DECLARE
    account public.user;
BEGIN
    SELECT * INTO account FROM public.user LIMIT 1;
    RETURN account.name;
END;
$$;

-- Function with non-reserved type name (should NOT be affected)
CREATE OR REPLACE FUNCTION count_users()
RETURNS integer
LANGUAGE plpgsql
AS $$
DECLARE
    total_count integer;
BEGIN
    SELECT count(*)::integer INTO total_count FROM public."user";
    RETURN total_count;
END;
$$;
