CREATE PROCEDURE update_user_status(
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