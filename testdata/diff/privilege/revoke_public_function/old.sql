CREATE FUNCTION get_user_data(user_id integer)
RETURNS text
LANGUAGE sql
AS $$ SELECT 'user_' || user_id::text; $$;
