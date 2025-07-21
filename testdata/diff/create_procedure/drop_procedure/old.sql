CREATE PROCEDURE cleanup_old_data(
    days_old integer DEFAULT 30
)
LANGUAGE plpgsql
AS $$
BEGIN
    DELETE FROM audit_logs 
    WHERE created_at < NOW() - INTERVAL '1 day' * days_old;
    
    DELETE FROM temp_data 
    WHERE created_at < NOW() - INTERVAL '1 day' * days_old;
    
    COMMIT;
END;
$$;