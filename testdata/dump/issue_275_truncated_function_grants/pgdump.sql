--
-- PostgreSQL database dump
--

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

-- Create test role
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'api_role') THEN
        CREATE ROLE api_role;
    END IF;
END $$;

--
-- Name: process_user_data(uuid, text, text, boolean); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.process_user_data(user_id uuid, user_name text, user_email text, is_active boolean) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    -- no-op
END;
$$;

--
-- Name: FUNCTION process_user_data(user_id uuid, user_name text, user_email text, is_active boolean); Type: ACL; Schema: public; Owner: -
--

GRANT EXECUTE ON FUNCTION public.process_user_data(user_id uuid, user_name text, user_email text, is_active boolean) TO api_role;

--
-- PostgreSQL database dump complete
--
