CREATE MATERIALIZED VIEW IF NOT EXISTS active_employees AS
 SELECT
    id,
    name,
    salary
   FROM employees
  WHERE status = 'active';
