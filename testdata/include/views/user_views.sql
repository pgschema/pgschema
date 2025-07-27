CREATE VIEW user_summary AS
SELECT 
    u.id,
    u.name,
    COUNT(o.id) as order_count
FROM users u
LEFT JOIN orders o ON u.id = o.user_id
GROUP BY u.id, u.name;

CREATE VIEW order_details AS
SELECT 
    o.id,
    o.status,
    u.name as user_name
FROM orders o
JOIN users u ON o.user_id = u.id;

COMMENT ON VIEW user_summary IS 'User order summary';

COMMENT ON VIEW order_details IS 'Order details with user info';