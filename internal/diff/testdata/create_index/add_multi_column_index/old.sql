CREATE TABLE public.employees (
    id integer NOT NULL,
    department_id integer,
    salary numeric(10,2),
    hire_date date,
    name text
);