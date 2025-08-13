ALTER TABLE dept_emp DROP CONSTRAINT dept_emp_dept_no_fkey;

ALTER TABLE dept_emp
ADD CONSTRAINT dept_emp_dept_no_fkey FOREIGN KEY (dept_no) REFERENCES department(dept_no) ON DELETE CASCADE;