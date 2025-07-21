CREATE TABLE public.employees (
    id integer NOT NULL,
    department_id integer,
    salary numeric(10,2),
    hire_date date,
    name text
);

CREATE INDEX idx_dept_salary_hire ON public.employees USING btree (department_id, salary DESC, hire_date);