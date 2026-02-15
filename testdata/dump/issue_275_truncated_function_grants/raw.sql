--
-- Test case for GitHub issue #275: Truncated functions in grants
--
-- This test case reproduces a bug where function signatures in GRANT EXECUTE
-- statements are truncated to 63 characters (PostgreSQL name type limit).
--
-- The function signature "process_user_data(user_id uuid, user_name text, user_email text, is_active boolean)"
-- gets truncated to 63 chars without the fix.
--

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'api_role') THEN
        CREATE ROLE api_role;
    END IF;
END $$;

CREATE FUNCTION process_user_data(user_id uuid, user_name text, user_email text, is_active boolean)
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
    -- no-op
END;
$$;

GRANT EXECUTE ON FUNCTION process_user_data(uuid, text, text, boolean) TO api_role;
