CREATE OR REPLACE TRIGGER trg_user_emails_insert
    INSTEAD OF INSERT ON user_emails
    FOR EACH ROW
    EXECUTE FUNCTION insert_user_emails();
