DROP PROCEDURE IF EXISTS simple_salary_update(integer, integer);

DROP TABLE IF EXISTS title CASCADE;

DROP TABLE IF EXISTS dept_manager CASCADE;

CREATE TYPE employee_status AS ENUM (
    'active',
    'inactive',
    'terminated'
);

CREATE TABLE employee_status_log (
    id SERIAL PRIMARY KEY,
    emp_no integer NOT NULL REFERENCES employee(emp_no) ON DELETE CASCADE,
    status employee_status NOT NULL,
    effective_date date DEFAULT CURRENT_DATE NOT NULL,
    notes text
);

CREATE INDEX idx_employee_status_log_effective_date ON employee_status_log (effective_date);

CREATE INDEX idx_employee_status_log_emp_no ON employee_status_log (emp_no);

ALTER TABLE audit ENABLE ROW LEVEL SECURITY;

CREATE POLICY audit_insert_system ON audit FOR INSERT TO PUBLIC WITH CHECK (true);

CREATE POLICY audit_user_isolation ON audit TO PUBLIC USING (user_name = CURRENT_USER);

ALTER TABLE employee ADD COLUMN status employee_status DEFAULT 'active' NOT NULL;