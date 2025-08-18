ALTER TABLE department
ADD CONSTRAINT department_dept_name_key UNIQUE (dept_name);

ALTER TABLE dept_emp DROP CONSTRAINT dept_emp_dept_no_fkey;

ALTER TABLE dept_emp
ADD CONSTRAINT dept_emp_dept_no_fkey FOREIGN KEY (dept_no) REFERENCES department(dept_no) ON DELETE CASCADE NOT VALID;

ALTER TABLE dept_emp VALIDATE CONSTRAINT dept_emp_dept_no_fkey;

ALTER TABLE dept_emp DROP CONSTRAINT dept_emp_emp_no_fkey;

ALTER TABLE dept_emp
ADD CONSTRAINT dept_emp_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES employee(emp_no) ON DELETE CASCADE NOT VALID;

ALTER TABLE dept_emp VALIDATE CONSTRAINT dept_emp_emp_no_fkey;

ALTER TABLE dept_manager DROP CONSTRAINT dept_manager_dept_no_fkey;

ALTER TABLE dept_manager
ADD CONSTRAINT dept_manager_dept_no_fkey FOREIGN KEY (dept_no) REFERENCES department(dept_no) ON DELETE CASCADE NOT VALID;

ALTER TABLE dept_manager VALIDATE CONSTRAINT dept_manager_dept_no_fkey;

ALTER TABLE dept_manager DROP CONSTRAINT dept_manager_emp_no_fkey;

ALTER TABLE dept_manager
ADD CONSTRAINT dept_manager_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES employee(emp_no) ON DELETE CASCADE NOT VALID;

ALTER TABLE dept_manager VALIDATE CONSTRAINT dept_manager_emp_no_fkey;

ALTER TABLE employee
ADD CONSTRAINT employee_gender_check CHECK (gender IN ('M', 'F')) NOT VALID;

ALTER TABLE employee VALIDATE CONSTRAINT employee_gender_check;

CREATE INDEX IF NOT EXISTS idx_employee_hire_date ON employee (hire_date);

ALTER TABLE salary DROP CONSTRAINT salary_emp_no_fkey;

ALTER TABLE salary
ADD CONSTRAINT salary_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES employee(emp_no) ON DELETE CASCADE NOT VALID;

ALTER TABLE salary VALIDATE CONSTRAINT salary_emp_no_fkey;

CREATE INDEX IF NOT EXISTS idx_salary_amount ON salary (amount);

ALTER TABLE title DROP CONSTRAINT title_emp_no_fkey;

ALTER TABLE title
ADD CONSTRAINT title_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES employee(emp_no) ON DELETE CASCADE NOT VALID;

ALTER TABLE title VALIDATE CONSTRAINT title_emp_no_fkey;
