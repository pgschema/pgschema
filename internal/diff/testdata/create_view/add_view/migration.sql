CREATE OR REPLACE VIEW public.active_employees AS
SELECT 
    id,
    name,
    salary
FROM public.employees
WHERE status = 'active';