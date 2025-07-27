CREATE PROCEDURE cleanup_orders()
LANGUAGE SQL
AS $$
    DELETE FROM orders WHERE status = 'completed';
$$;

CREATE PROCEDURE update_status(user_id_param INTEGER, new_status TEXT)
LANGUAGE SQL
AS $$
    UPDATE orders SET status = new_status WHERE user_id = user_id_param;
$$;