CREATE TABLE public.employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    email VARCHAR(100),
    bio TEXT,
    status VARCHAR(20) NOT NULL,
    department_id INTEGER,
    priority INTEGER
);

CREATE TABLE public.departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    manager_id INTEGER
);

-- View testing array operators: all ANY/ALL operators are preserved
CREATE VIEW public.array_operators_view AS
SELECT
    id,
    priority,
    -- All ANY operations preserve the ANY syntax
    CASE WHEN priority = ANY(ARRAY[10, 20, 30]) THEN 'matched' ELSE 'not_matched' END AS equal_any_test,
    CASE WHEN priority > ANY(ARRAY[10, 20, 30]) THEN 'high' ELSE 'low' END AS greater_any_test,
    CASE WHEN priority < ANY(ARRAY[5, 15, 25]) THEN 'found_lower' ELSE 'all_higher' END AS less_any_test,
    CASE WHEN priority <> ANY(ARRAY[1, 2, 3]) THEN 'different' ELSE 'same' END AS not_equal_any_test
FROM employees;

-- View testing COALESCE, string concatenation, and to_tsvector for full text search
CREATE VIEW public.text_search_view AS
SELECT
    id,
    COALESCE(first_name || ' ' || last_name, 'Anonymous') AS display_name,
    COALESCE(email, '') AS email,
    COALESCE(bio, 'No description available') AS description,
    to_tsvector('english', COALESCE(first_name, '') || ' ' || COALESCE(last_name, '') || ' ' || COALESCE(bio, '')) AS search_vector
FROM employees
WHERE status = 'active';

-- View testing NULLIF, GREATEST, LEAST, and joins (regression test for issue #103)
CREATE VIEW public.nullif_functions_view AS
SELECT
    e.id,
    e.name AS employee_name,
    d.name AS department_name,
    -- NULLIF to avoid divide-by-zero (main issue from #103)
    (e.priority - d.manager_id) / NULLIF(d.manager_id, 0) AS priority_ratio,
    -- Multiple NULLIF expressions
    NULLIF(e.status, 'inactive') AS active_status,
    NULLIF(e.email, '') AS valid_email,
    -- GREATEST and LEAST functions
    GREATEST(e.priority, 0) AS min_priority,
    LEAST(e.priority, 100) AS max_priority,
    GREATEST(e.id, d.id, e.department_id) AS max_id,
    -- Complex CASE with NULLIF
    CASE
        WHEN NULLIF(e.department_id, 0) IS NOT NULL THEN 'assigned'
        ELSE 'unassigned'
    END AS assignment_status
FROM employees e
JOIN departments d USING (id)
WHERE e.priority > 0;