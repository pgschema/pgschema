CREATE OR REPLACE VIEW employee_view AS
SELECT 
    e.id,
    e.name,
    e.department,
    e.salary
FROM public.employees e
WHERE e.active = true;

COMMENT ON VIEW employee_view IS 'Shows all active employees';
