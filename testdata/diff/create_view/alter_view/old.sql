CREATE TABLE public.employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    salary DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) NOT NULL
);

CREATE VIEW public.active_employees AS
SELECT 
    status,
    COUNT(*) as employee_count,
    AVG(salary) as avg_salary
FROM employees
WHERE status = 'active'
GROUP BY status
HAVING COUNT(*) > 0
ORDER BY avg_salary DESC;