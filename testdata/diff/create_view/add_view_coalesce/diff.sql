CREATE OR REPLACE VIEW user_search_view AS
 SELECT
    u.id,
    COALESCE(p.display_name, u.first_name || ' ' || u.last_name) AS display_name,
    COALESCE(u.email, '') AS email,
    COALESCE(p.description, u.bio, 'No description available') AS description,
    to_tsvector('english', COALESCE(p.display_name, '') || ' ' || COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '') || ' ' || COALESCE(p.description, '')) AS search_vector
   FROM users u
     LEFT JOIN profiles p ON u.id = p.user_id
  WHERE u.status = 'active';