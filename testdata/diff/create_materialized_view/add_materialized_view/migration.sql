CREATE MATERIALIZED VIEW active_employees AS
 SELECT
    id,
    name,
    salary
   FROM employees
  WHERE status = 'active';