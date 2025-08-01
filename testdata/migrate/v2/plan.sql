ALTER TABLE department
ADD CONSTRAINT department_dept_name_key UNIQUE (dept_name);

ALTER TABLE employee
ADD CONSTRAINT employee_gender_check CHECK (gender IN ('M', 'F'));

CREATE INDEX idx_employee_hire_date ON employee (hire_date);

CREATE INDEX idx_salary_amount ON salary (amount);