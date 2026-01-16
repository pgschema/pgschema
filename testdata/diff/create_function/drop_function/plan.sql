REVOKE EXECUTE ON FUNCTION process_order(order_id integer, discount_percent numeric) FROM api_role;

DROP FUNCTION IF EXISTS process_payment(integer, text);

DROP FUNCTION IF EXISTS process_order(integer, numeric);

DROP FUNCTION IF EXISTS get_user_stats(integer);
