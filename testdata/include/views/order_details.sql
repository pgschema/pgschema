--
-- Name: order_details; Type: VIEW; Schema: -; Owner: -
--

CREATE OR REPLACE VIEW order_details AS
 SELECT o.id,
    o.status,
    u.name AS user_name
   FROM orders o
     JOIN users u ON o.user_id = u.id;

COMMENT ON VIEW order_details IS 'Order details with user info';