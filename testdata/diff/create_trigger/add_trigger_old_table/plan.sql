CREATE OR REPLACE TRIGGER orders_delete_trigger
    AFTER DELETE ON orders
    REFERENCING OLD TABLE AS old_orders
    FOR EACH STATEMENT
    EXECUTE FUNCTION archive_deleted_orders();
