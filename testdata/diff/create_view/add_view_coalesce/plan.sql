CREATE OR REPLACE VIEW user_search_view AS
 SELECT
    id,
    COALESCE(first_name || ' ' || last_name, 'Anonymous') AS display_name,
    COALESCE(email, '') AS email,
    COALESCE(bio, 'No description available') AS description,
    to_tsvector('english', COALESCE(first_name, '') || ' ' || COALESCE(last_name, '') || ' ' || COALESCE(bio, '')) AS search_vector
   FROM users
  WHERE status = 'active';
