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