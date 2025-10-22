CREATE OR REPLACE VIEW array_operators_view AS
 SELECT id,
    priority,
    CASE WHEN priority IN (10, 20, 30) THEN 'matched' ELSE 'not_matched' END AS equal_any_test,
    CASE WHEN priority > ANY (ARRAY[10, 20, 30]) THEN 'high' ELSE 'low' END AS greater_any_test,
    CASE WHEN priority < ANY (ARRAY[5, 15, 25]) THEN 'found_lower' ELSE 'all_higher' END AS less_any_test,
    CASE WHEN priority <> ANY (ARRAY[1, 2, 3]) THEN 'different' ELSE 'same' END AS not_equal_any_test
   FROM employees;

CREATE OR REPLACE VIEW text_search_view AS
 SELECT id,
    COALESCE((first_name::text || ' '::text) || last_name::text, 'Anonymous'::text) AS display_name,
    COALESCE(email, ''::character varying) AS email,
    COALESCE(bio, 'No description available'::text) AS description,
    to_tsvector('english'::regconfig, (((COALESCE(first_name, ''::character varying)::text || ' '::text) || COALESCE(last_name, ''::character varying)::text) || ' '::text) || COALESCE(bio, ''::text)) AS search_vector
   FROM employees
  WHERE status::text = 'active'::text;
