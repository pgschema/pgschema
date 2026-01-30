DROP VIEW IF EXISTS employee_summary CASCADE;

DROP MATERIALIZED VIEW active_employees RESTRICT;

CREATE MATERIALIZED VIEW IF NOT EXISTS active_employees AS
 SELECT id,
    name,
    salary,
    'active'::text AS status_label
   FROM employees
  WHERE status::text = 'active'::text;

CREATE OR REPLACE VIEW employee_summary AS
 SELECT id,
    name
   FROM active_employees;
