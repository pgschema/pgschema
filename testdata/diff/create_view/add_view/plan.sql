CREATE OR REPLACE VIEW employee_department_view AS
 SELECT
    e.id,
    e.name AS employee_name,
    d.name AS department_name,
    d.manager_id
   FROM employees e
     JOIN departments d ON e.department_id = d.id
  WHERE e.name IS NOT NULL AND d.manager_id IS NOT NULL;
