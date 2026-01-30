CREATE TABLE public.employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    salary DECIMAL(10,2) NOT NULL,
    department VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL
);

-- First materialized view - modified with new column (requires drop/recreate)
CREATE MATERIALIZED VIEW public.active_employees AS
SELECT
    id,
    name,
    salary,
    'active' AS status_label  -- New column forces drop/recreate
FROM employees
WHERE status = 'active';

-- Second materialized view - modified with new column (requires drop/recreate)
CREATE MATERIALIZED VIEW public.dept_stats AS
SELECT
    department,
    COUNT(*) AS employee_count,
    AVG(salary) AS avg_salary  -- New column forces drop/recreate
FROM employees
GROUP BY department;

-- View V1: depends on materialized view (for nested dependency test)
CREATE VIEW public.employee_names AS
SELECT
    id,
    name
FROM active_employees;

-- View V2: nested dependency - depends on V1 which depends on mat view
CREATE VIEW public.employee_ids AS
SELECT
    id
FROM employee_names;

-- View that depends on BOTH materialized views (multi-matview dependency test)
CREATE VIEW public.employee_summary AS
SELECT
    ae.id,
    ae.name,
    ds.employee_count AS dept_size
FROM active_employees ae
CROSS JOIN dept_stats ds
LIMIT 10;
