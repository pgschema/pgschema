CREATE OR REPLACE TRIGGER trg_user_emails_insert
    INSTEAD OF INSERT ON user_emails
    FOR EACH ROW
    EXECUTE FUNCTION insert_user_emails();

CREATE OR REPLACE FUNCTION insert_user_emails()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    INSERT INTO users (email)
    VALUES (NEW.email)
    RETURNING id, email INTO NEW.id, NEW.email;
    RETURN NEW;
END;
$$;
