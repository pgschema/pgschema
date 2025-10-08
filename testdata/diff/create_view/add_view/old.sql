CREATE TABLE public.employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    department_id INTEGER
);

CREATE TABLE public.departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    manager_id INTEGER
);