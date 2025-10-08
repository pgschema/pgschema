CREATE TABLE public.departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    location VARCHAR(100)
);

CREATE TABLE public.employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    department_id INTEGER REFERENCES departments(id),
    salary DECIMAL(10,2) NOT NULL
);

CREATE VIEW public.employee_department_view AS
SELECT
    e.id as employee_id,
    e.name as employee_name,
    d.name as department_name,
    d.location
FROM employees e
INNER JOIN departments d ON e.department_id = d.id;

CREATE VIEW public.all_employees_with_dept AS
SELECT
    e.id,
    e.name,
    d.name as dept_name
FROM employees e
LEFT JOIN departments d ON e.department_id = d.id;

CREATE VIEW public.all_departments_with_emp AS
SELECT
    d.id,
    d.name as dept_name,
    e.name as emp_name
FROM employees e
RIGHT JOIN departments d ON e.department_id = d.id;

CREATE VIEW public.complete_employee_dept AS
SELECT
    e.id as emp_id,
    e.name as emp_name,
    d.id as dept_id,
    d.name as dept_name
FROM employees e
FULL OUTER JOIN departments d ON e.department_id = d.id;

CREATE VIEW public.employee_dept_cross AS
SELECT
    e.name as employee_name,
    d.name as department_name
FROM employees e
CROSS JOIN departments d;
