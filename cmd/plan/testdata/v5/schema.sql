--
-- Name: employee_status; Type: TYPE; Schema: -; Owner: -
--

CREATE TYPE employee_status AS ENUM ('active', 'inactive', 'terminated');


--
-- Name: log_dml_operations; Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION log_dml_operations()
RETURNS trigger
LANGUAGE PLPGSQL
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




--
-- Name: audit; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE audit (
    id SERIAL NOT NULL,
    operation text NOT NULL,
    query text,
    user_name text NOT NULL,
    changed_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);


--
-- Name: idx_audit_changed_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_audit_changed_at ON audit (changed_at);


--
-- Name: idx_audit_operation; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_audit_operation ON audit (operation);


--
-- Name: idx_audit_username; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_audit_username ON audit (user_name);


--
-- Name: audit; Type: TABLE; Schema: -; Owner: -
--

ALTER TABLE audit ENABLE ROW LEVEL SECURITY;


--
-- Name: audit_insert_system; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY audit_insert_system ON audit FOR INSERT TO public WITH CHECK (true);


--
-- Name: audit_user_isolation; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY audit_user_isolation ON audit TO public USING ((user_name = CURRENT_USER));


--
-- Name: department; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE department (
    dept_no text NOT NULL,
    dept_name text NOT NULL,
    PRIMARY KEY (dept_no),
    UNIQUE (dept_name)
);


--
-- Name: employee; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE employee (
    emp_no SERIAL NOT NULL,
    birth_date date NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    gender text NOT NULL CHECK (gender IN('M', 'F')),
    hire_date date NOT NULL,
    status employee_status NOT NULL DEFAULT 'active',
    PRIMARY KEY (emp_no)
);


--
-- Name: idx_employee_hire_date; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_employee_hire_date ON employee (hire_date);


--
-- Name: dept_emp; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE dept_emp (
    emp_no integer NOT NULL,
    dept_no text NOT NULL,
    from_date date NOT NULL,
    to_date date NOT NULL,
    PRIMARY KEY (emp_no, dept_no),
    FOREIGN KEY (dept_no) REFERENCES department (dept_no) ON DELETE CASCADE,
    FOREIGN KEY (emp_no) REFERENCES employee (emp_no) ON DELETE CASCADE
);




--
-- Name: salary; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE salary (
    emp_no integer NOT NULL,
    amount integer NOT NULL,
    from_date date NOT NULL,
    to_date date NOT NULL,
    PRIMARY KEY (emp_no, from_date),
    FOREIGN KEY (emp_no) REFERENCES employee (emp_no) ON DELETE CASCADE
);


--
-- Name: idx_salary_amount; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_salary_amount ON salary (amount);


--
-- Name: salary_log_trigger; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER salary_log_trigger
    AFTER UPDATE OR DELETE ON salary
    FOR EACH ROW
    EXECUTE FUNCTION log_dml_operations('payroll', 'high');


--
-- Name: employee_status_log_trigger; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER employee_status_log_trigger
    AFTER INSERT OR UPDATE ON employee_status_log
    FOR EACH ROW
    EXECUTE FUNCTION log_dml_operations('hr', 'medium');


--
-- Name: employee_status_log; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE employee_status_log (
    id SERIAL NOT NULL,
    emp_no integer NOT NULL,
    status employee_status NOT NULL,
    effective_date date NOT NULL DEFAULT CURRENT_DATE,
    notes text,
    PRIMARY KEY (id),
    FOREIGN KEY (emp_no) REFERENCES employee (emp_no) ON DELETE CASCADE
);


--
-- Name: idx_employee_status_log_emp_no; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_employee_status_log_emp_no ON employee_status_log (emp_no);


--
-- Name: idx_employee_status_log_effective_date; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_employee_status_log_effective_date ON employee_status_log (effective_date);


--
-- Name: dept_emp_latest_date; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW dept_emp_latest_date AS
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM dept_emp
  GROUP BY emp_no;


--
-- Name: current_dept_emp; Type: VIEW; Schema: -; Owner: -
--

CREATE VIEW current_dept_emp AS
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (dept_emp d
     JOIN dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));