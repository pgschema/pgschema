

CREATE OR REPLACE FUNCTION log_dml_operations()
RETURNS trigger
LANGUAGE plpgsql
SECURITY INVOKER
VOLATILE
AS $$
DECLARE
    table_category TEXT;
    log_level TEXT;
BEGIN
    -- Get arguments passed from trigger (if any)
    -- TG_ARGV[0] is the first argument, TG_ARGV[1] is the second
    table_category := COALESCE(TG_ARGV[0], 'default');
    log_level := COALESCE(TG_ARGV[1], 'standard');
    
    IF (TG_OP = 'INSERT') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (
            'INSERT [' || table_category || ':' || log_level || ']', 
            current_query(), 
            current_user
        );
        RETURN NEW;
    ELSIF (TG_OP = 'UPDATE') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (
            'UPDATE [' || table_category || ':' || log_level || ']', 
            current_query(), 
            current_user
        );
        RETURN NEW;
    ELSIF (TG_OP = 'DELETE') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (
            'DELETE [' || table_category || ':' || log_level || ']', 
            current_query(), 
            current_user
        );
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$;


CREATE OR REPLACE PROCEDURE simple_salary_update(
    p_emp_no integer,
    p_amount integer
)
LANGUAGE plpgsql
AS $$
BEGIN
    -- Simple update of salary amount
    UPDATE salary 
    SET amount = p_amount 
    WHERE emp_no = p_emp_no 
    AND to_date = '9999-01-01';
    
    RAISE NOTICE 'Updated salary for employee % to $%', p_emp_no, p_amount;
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


CREATE INDEX idx_audit_operation ON audit (operation);


CREATE INDEX idx_audit_username ON audit (user_name);


CREATE OR REPLACE VIEW dept_emp_latest_date AS
SELECT 
    emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
FROM dept_emp GROUP BY emp_no;


CREATE OR REPLACE VIEW current_dept_emp AS
SELECT 
    l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
FROM dept_emp d JOIN dept_emp_latest_date l ON d.emp_no = l.emp_no AND d.from_date = l.from_date AND l.to_date = d.to_date;


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
    EXECUTE FUNCTION log_dml_operations('payroll', 'high');


ALTER TABLE title DROP CONSTRAINT title_pkey;


ALTER TABLE public.title 
ADD CONSTRAINT title_emp_no_pkey PRIMARY KEY (emp_no, title, from_date);
