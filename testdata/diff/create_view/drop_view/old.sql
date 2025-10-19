CREATE TABLE public.employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    salary DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) NOT NULL
);

CREATE VIEW public.active_employees AS
SELECT 
    id,
    name,
    salary
FROM employees
WHERE status = 'active';