

CREATE OR REPLACE FUNCTION log_dml_operations()
RETURNS trigger
LANGUAGE plpgsql
SECURITY INVOKER
VOLATILE
AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES ('INSERT', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = 'UPDATE') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES ('UPDATE', current_query(), current_user);
        RETURN NEW;
    ELSIF (TG_OP = 'DELETE') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES ('DELETE', current_query(), current_user);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$;


CREATE TABLE audit (
    id SERIAL NOT NULL,
    operation text NOT NULL,
    query text,
    user_name text NOT NULL,
    changed_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);


CREATE INDEX idx_audit_changed_at ON audit (changed_at);


ALTER TABLE department DROP CONSTRAINT department_pkey;


ALTER TABLE department 
ADD CONSTRAINT department_dept_name_key UNIQUE (dept_name);


ALTER TABLE public.department 
ADD CONSTRAINT department_dept_no_pkey PRIMARY KEY (dept_no);


ALTER TABLE employee DROP CONSTRAINT employee_pkey;


ALTER TABLE employee ALTER COLUMN emp_no SET DEFAULT nextval('public.employee_emp_no_seq');


ALTER TABLE public.employee 
ADD CONSTRAINT employee_emp_no_pkey PRIMARY KEY (emp_no);


ALTER TABLE employee 
ADD CONSTRAINT employee_gender_check CHECK (gender IN ('M', 'F'));


CREATE INDEX idx_employee_hire_date ON employee (hire_date);


ALTER TABLE dept_emp DROP CONSTRAINT dept_emp_pkey;


ALTER TABLE public.dept_emp 
ADD CONSTRAINT dept_emp_emp_no_pkey PRIMARY KEY (emp_no, dept_no);


ALTER TABLE dept_manager DROP CONSTRAINT dept_manager_pkey;


ALTER TABLE public.dept_manager 
ADD CONSTRAINT dept_manager_emp_no_pkey PRIMARY KEY (emp_no, dept_no);


ALTER TABLE salary DROP CONSTRAINT salary_pkey;


ALTER TABLE public.salary 
ADD CONSTRAINT salary_emp_no_pkey PRIMARY KEY (emp_no, from_date);


CREATE INDEX idx_salary_amount ON salary (amount);


CREATE OR REPLACE TRIGGER salary_log_trigger
    AFTER UPDATE OR DELETE ON salary
    FOR EACH ROW
    EXECUTE FUNCTION log_dml_operations();


ALTER TABLE title DROP CONSTRAINT title_pkey;


ALTER TABLE public.title 
ADD CONSTRAINT title_emp_no_pkey PRIMARY KEY (emp_no, title, from_date);
