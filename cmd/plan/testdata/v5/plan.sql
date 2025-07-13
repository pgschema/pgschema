DROP PROCEDURE IF EXISTS simple_salary_update(integer, integer);


DROP TABLE IF EXISTS title CASCADE;


DROP TABLE IF EXISTS dept_manager CASCADE;


CREATE TYPE employee_status AS ENUM (
    'active',
    'inactive',
    'terminated'
);


CREATE TABLE employee_status_log (
    id SERIAL NOT NULL,
    emp_no integer NOT NULL,
    status employee_status NOT NULL,
    effective_date date DEFAULT CURRENT_DATE NOT NULL,
    notes text,
    PRIMARY KEY (id),
    FOREIGN KEY (emp_no) REFERENCES employee (emp_no) ON DELETE CASCADE
);


CREATE INDEX idx_employee_status_log_effective_date ON employee_status_log (effective_date);


CREATE INDEX idx_employee_status_log_emp_no ON employee_status_log (emp_no);


ALTER TABLE employee ADD COLUMN status employee_status DEFAULT 'active' NOT NULL;


ALTER TABLE employee ALTER COLUMN emp_no SET DEFAULT nextval('public.employee_emp_no_seq');


CREATE OR REPLACE TRIGGER salary_log_trigger
    AFTER UPDATE OR DELETE ON salary
    FOR EACH ROW
    EXECUTE FUNCTION log_dml_operations('payroll', 'high');


ALTER TABLE audit ALTER COLUMN id SET DEFAULT nextval('public.audit_id_seq');


ALTER TABLE audit ENABLE ROW LEVEL SECURITY;


CREATE POLICY audit_insert_system ON audit FOR INSERT TO PUBLIC WITH CHECK (true);


CREATE POLICY audit_user_isolation ON audit TO PUBLIC USING ((user_name = CURRENT_USER));


CREATE OR REPLACE VIEW current_dept_emp AS
SELECT 
    l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
FROM dept_emp d JOIN dept_emp_latest_date l ON d.emp_no = l.emp_no AND d.from_date = l.from_date AND l.to_date = d.to_date;


CREATE OR REPLACE VIEW dept_emp_latest_date AS
SELECT 
    emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
FROM dept_emp GROUP BY emp_no;


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