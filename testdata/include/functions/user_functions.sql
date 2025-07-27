CREATE FUNCTION get_user_count()
RETURNS INTEGER AS $$
    SELECT COUNT(*) FROM users;
$$ LANGUAGE SQL;

CREATE FUNCTION get_order_count(user_id_param INTEGER)
RETURNS INTEGER AS $$
    SELECT COUNT(*) FROM orders WHERE user_id = user_id_param;
$$ LANGUAGE SQL;