--
-- Test case for GitHub issue #252: Schema qualifiers in function bodies
--
-- This test reproduces the inconsistency where function bodies retain
-- schema-qualified references (e.g., public.users) while table definitions
-- are dumped without schema qualification.
--
-- The fix strips the current schema qualifier from function/procedure bodies
-- so the dump output is consistently unqualified.
--

CREATE TABLE users (
    id serial PRIMARY KEY,
    name text NOT NULL,
    email text
);

-- Function with schema-qualified table reference in body
CREATE OR REPLACE FUNCTION get_user_count()
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN (SELECT count(*)::integer FROM public.users);
END;
$$;

-- Function with multiple schema-qualified references
CREATE OR REPLACE FUNCTION get_user_by_name(p_name text)
RETURNS TABLE(id integer, name text, email text)
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY SELECT u.id, u.name, u.email FROM public.users u WHERE u.name = p_name;
END;
$$;

-- Function with string literal containing schema name (should NOT be modified)
CREATE OR REPLACE FUNCTION get_table_info()
RETURNS text
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN 'Table: public.users';
END;
$$;

-- Procedure with schema-qualified reference in body
CREATE OR REPLACE PROCEDURE insert_user(p_name text, p_email text)
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO public.users (name, email) VALUES (p_name, p_email);
END;
$$;
