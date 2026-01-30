CREATE TABLE public.employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    salary DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) NOT NULL
);

-- Modified materialized view with new column (requires drop and recreate)
CREATE MATERIALIZED VIEW public.active_employees AS
SELECT
    id,
    name,
    salary,
    'active' AS status_label  -- New column forces drop/recreate
FROM employees
WHERE status = 'active';

-- View that depends on the materialized view (reproduces issue #268)
CREATE VIEW public.employee_summary AS
SELECT
    id,
    name
FROM active_employees;
