CREATE OR REPLACE VIEW user_search_view AS
 SELECT id,
    COALESCE((first_name::text || ' '::text) || last_name::text, 'Anonymous'::text) AS display_name,
    COALESCE(email, ''::character varying) AS email,
    COALESCE(bio, 'No description available'::text) AS description,
    to_tsvector('english'::regconfig, (((COALESCE(first_name, ''::character varying)::text || ' '::text) || COALESCE(last_name, ''::character varying)::text) || ' '::text) || COALESCE(bio, ''::text)) AS search_vector
   FROM users
  WHERE status::text = 'active'::text;
