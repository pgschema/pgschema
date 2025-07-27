CREATE TABLE public.employees (
    id integer NOT NULL,
    name text NOT NULL,
    department text,
    salary numeric,
    active boolean DEFAULT true
);

CREATE VIEW public.employee_view AS
SELECT 
    e.id,
    e.name,
    e.department,
    e.salary
FROM public.employees e
WHERE e.active = true;

COMMENT ON VIEW public.employee_view IS 'Shows all active employees';