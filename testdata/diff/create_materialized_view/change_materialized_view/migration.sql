DROP MATERIALIZED VIEW active_employees;

CREATE MATERIALIZED VIEW active_employees AS
 SELECT
    id,
    name,
    salary,
    status
   FROM employees
  WHERE status = 'active';