CREATE OR REPLACE VIEW active_employees AS
 SELECT status,
    count(*) AS employee_count,
    avg(salary) AS avg_salary
   FROM employees
  WHERE status::text = 'active'::text
  GROUP BY status
 HAVING avg(salary) > 50000::numeric
  ORDER BY (count(*)), (avg(salary)) DESC;
