CREATE OR REPLACE VIEW active_employees AS
 SELECT
    id,
    name,
    salary
   FROM employees
  WHERE status = 'active';
