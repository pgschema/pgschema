CREATE OR REPLACE VIEW array_operators_view AS
 SELECT id,
    priority,
    CASE WHEN priority IN (10, 20, 30) THEN 'matched' ELSE 'not_matched' END AS equal_any_test,
    CASE WHEN priority > ANY (ARRAY[10, 20, 30]) THEN 'high' ELSE 'low' END AS greater_any_test,
    CASE WHEN priority < ANY (ARRAY[5, 15, 25]) THEN 'found_lower' ELSE 'all_higher' END AS less_any_test,
    CASE WHEN priority <> ANY (ARRAY[1, 2, 3]) THEN 'different' ELSE 'same' END AS not_equal_any_test
   FROM employees;

CREATE OR REPLACE VIEW nullif_functions_view AS
 SELECT e.id,
    e.name AS employee_name,
    d.name AS department_name,
    (e.priority - d.manager_id) / NULLIF(d.manager_id, 0) AS priority_ratio,
    NULLIF(e.status::text, 'inactive'::text) AS active_status,
    NULLIF(e.email::text, ''::text) AS valid_email,
    GREATEST(e.priority, 0) AS min_priority,
    LEAST(e.priority, 100) AS max_priority,
    GREATEST(e.id, d.id, e.department_id) AS max_id,
        CASE
            WHEN NULLIF(e.department_id, 0) IS NOT NULL THEN 'assigned'::text
            ELSE 'unassigned'::text
        END AS assignment_status
   FROM employees e
     JOIN departments d USING (id)
  WHERE e.priority > 0;

CREATE OR REPLACE VIEW text_search_view AS
 SELECT id,
    COALESCE((first_name::text || ' '::text) || last_name::text, 'Anonymous'::text) AS display_name,
    COALESCE(email, ''::character varying) AS email,
    COALESCE(bio, 'No description available'::text) AS description,
    to_tsvector('english'::regconfig, (((COALESCE(first_name, ''::character varying)::text || ' '::text) || COALESCE(last_name, ''::character varying)::text) || ' '::text) || COALESCE(bio, ''::text)) AS search_vector
   FROM employees
  WHERE status::text = 'active'::text;
