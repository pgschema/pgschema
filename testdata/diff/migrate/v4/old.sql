-- Version 3 Changes from v2:
-- - Added log_dml_operations() trigger function for DML auditing
-- - Added audit table to track INSERT/UPDATE/DELETE operations  
-- - Added idx_audit_changed_at index on audit table
-- - Added salary_log_trigger on salary table to log changes

--
-- Name: log_dml_operations; Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION log_dml_operations()
RETURNS trigger
LANGUAGE PLPGSQL
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
-- Name: dept_manager; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE dept_manager (
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
    EXECUTE FUNCTION log_dml_operations();


--
-- Name: title; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE title (
    emp_no integer NOT NULL,
    title text NOT NULL,
    from_date date NOT NULL,
    to_date date,
    PRIMARY KEY (emp_no, title, from_date),
    FOREIGN KEY (emp_no) REFERENCES employee (emp_no) ON DELETE CASCADE
);
