--
-- Name: user_summary; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW user_summary AS
 SELECT u.id,
    u.name,
    count(o.id) AS order_count
   FROM users u
     JOIN orders o ON u.id = o.user_id
  GROUP BY u.id, u.name;;

COMMENT ON VIEW user_summary IS 'User order summary';