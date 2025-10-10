CREATE CONSTRAINT TRIGGER prevent_code_update_trigger
    AFTER UPDATE ON products
    DEFERRABLE INITIALLY IMMEDIATE
    FOR EACH ROW
    EXECUTE FUNCTION prevent_code_update();
