--
-- PostgreSQL database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 0.1.2


--
-- Name: log_dml_operations(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE FUNCTION log_dml_operations() RETURNS trigger
    LANGUAGE plpgsql
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
    id SERIAL PRIMARY KEY,
    operation text NOT NULL,
    query text,
    user_name text NOT NULL,
    changed_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: department; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE department (
    dept_no text PRIMARY KEY,
    dept_name text NOT NULL UNIQUE
);


--
-- Name: dept_emp; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE dept_emp (
    emp_no integer NOT NULL,
    dept_no text NOT NULL,
    from_date date NOT NULL,
    to_date date NOT NULL
);


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


--
-- Name: dept_manager; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE dept_manager (
    emp_no integer NOT NULL,
    dept_no text NOT NULL,
    from_date date NOT NULL,
    to_date date NOT NULL
);


--
-- Name: employee; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE employee (
    emp_no SERIAL PRIMARY KEY,
    birth_date date NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    gender text NOT NULL,
    hire_date date NOT NULL,
    CONSTRAINT employee_gender_check CHECK ((gender = ANY (ARRAY['M'::text, 'F'::text])))
);


--
-- Name: salary; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE salary (
    emp_no integer NOT NULL,
    amount integer NOT NULL,
    from_date date NOT NULL,
    to_date date NOT NULL
);


--
-- Name: title; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE title (
    emp_no integer NOT NULL,
    title text NOT NULL,
    from_date date NOT NULL,
    to_date date
);


--
-- Name: audit audit_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY audit
    ADD CONSTRAINT audit_pkey PRIMARY KEY (id);


--
-- Name: department department_dept_name_key; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY department
    ADD CONSTRAINT department_dept_name_key UNIQUE (dept_name);


--
-- Name: department department_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY department
    ADD CONSTRAINT department_pkey PRIMARY KEY (dept_no);


--
-- Name: dept_emp dept_emp_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY dept_emp
    ADD CONSTRAINT dept_emp_pkey PRIMARY KEY (emp_no, dept_no);


--
-- Name: dept_manager dept_manager_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY dept_manager
    ADD CONSTRAINT dept_manager_pkey PRIMARY KEY (emp_no, dept_no);


--
-- Name: employee employee_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY employee
    ADD CONSTRAINT employee_pkey PRIMARY KEY (emp_no);


--
-- Name: salary salary_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY salary
    ADD CONSTRAINT salary_pkey PRIMARY KEY (emp_no, from_date);


--
-- Name: title title_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY title
    ADD CONSTRAINT title_pkey PRIMARY KEY (emp_no, title, from_date);


--
-- Name: idx_audit_changed_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_audit_changed_at ON audit USING btree (changed_at);


--
-- Name: idx_audit_operation; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_audit_operation ON audit USING btree (operation);


--
-- Name: idx_audit_username; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_audit_username ON audit USING btree (user_name);


--
-- Name: idx_employee_hire_date; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_employee_hire_date ON employee USING btree (hire_date);


--
-- Name: idx_salary_amount; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_salary_amount ON salary USING btree (amount);


--
-- Name: salary salary_log_trigger; Type: TRIGGER; Schema: -; Owner: -
--

CREATE TRIGGER salary_log_trigger AFTER UPDATE OR DELETE ON salary FOR EACH ROW EXECUTE FUNCTION log_dml_operations();


--
-- Name: dept_emp dept_emp_dept_no_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY dept_emp
    ADD CONSTRAINT dept_emp_dept_no_fkey FOREIGN KEY (dept_no) REFERENCES department(dept_no) ON DELETE CASCADE;


--
-- Name: dept_emp dept_emp_emp_no_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY dept_emp
    ADD CONSTRAINT dept_emp_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES employee(emp_no) ON DELETE CASCADE;


--
-- Name: dept_manager dept_manager_dept_no_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY dept_manager
    ADD CONSTRAINT dept_manager_dept_no_fkey FOREIGN KEY (dept_no) REFERENCES department(dept_no) ON DELETE CASCADE;


--
-- Name: dept_manager dept_manager_emp_no_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY dept_manager
    ADD CONSTRAINT dept_manager_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES employee(emp_no) ON DELETE CASCADE;


--
-- Name: salary salary_emp_no_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY salary
    ADD CONSTRAINT salary_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES employee(emp_no) ON DELETE CASCADE;


--
-- Name: title title_emp_no_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY title
    ADD CONSTRAINT title_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES employee(emp_no) ON DELETE CASCADE;


--
-- Name: audit; Type: ROW SECURITY; Schema: -; Owner: -
--

ALTER TABLE audit ENABLE ROW LEVEL SECURITY;


--
-- Name: audit audit_insert_system; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY audit_insert_system ON audit FOR INSERT WITH CHECK (true);


--
-- Name: audit audit_user_isolation; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY audit_user_isolation ON audit USING ((user_name = CURRENT_USER));


--
-- PostgreSQL database dump complete
--

