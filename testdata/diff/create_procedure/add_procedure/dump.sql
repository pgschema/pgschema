--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: update_user_status; Type: PROCEDURE; Schema: -; Owner: -
--

CREATE OR REPLACE PROCEDURE update_user_status(
    user_id integer,
    new_status text
)
LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE users 
    SET status = new_status, updated_at = NOW() 
    WHERE id = user_id;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'User not found: %', user_id;
    END IF;
    
    COMMIT;
END;
$$;

