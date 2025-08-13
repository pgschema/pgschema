CREATE INDEX IF NOT EXISTS idx_dept_salary_hire ON employees (department_id, salary DESC, hire_date);
