CREATE OR REPLACE TRIGGER products_description_trigger
    BEFORE UPDATE ON products
    FOR EACH ROW
    WHEN (NEW.description IS DISTINCT FROM OLD.description)
    EXECUTE FUNCTION log_description_change();

CREATE OR REPLACE TRIGGER products_status_trigger
    BEFORE UPDATE ON products
    FOR EACH ROW
    WHEN (NEW.status IS NOT DISTINCT FROM OLD.status)
    EXECUTE FUNCTION skip_status_change();
