CREATE OR REPLACE VIEW active_employees AS
 SELECT
    id,
    name,
    salary,
    status
   FROM employees
  WHERE status = 'active';
