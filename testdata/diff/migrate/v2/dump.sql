--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: department; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS department (
    dept_no text PRIMARY KEY,
    dept_name text NOT NULL UNIQUE
);

--
-- Name: employee; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS employee (
    emp_no SERIAL PRIMARY KEY,
    birth_date date NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    gender text NOT NULL CHECK (gender IN ('M', 'F')),
    hire_date date NOT NULL
);

--
-- Name: idx_employee_hire_date; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_employee_hire_date ON employee (hire_date);

--
-- Name: dept_emp; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS dept_emp (
    emp_no integer REFERENCES employee(emp_no) ON DELETE CASCADE,
    dept_no text REFERENCES department(dept_no) ON DELETE CASCADE,
    from_date date NOT NULL,
    to_date date NOT NULL,
    PRIMARY KEY (emp_no, dept_no)
);

--
-- Name: dept_manager; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS dept_manager (
    emp_no integer REFERENCES employee(emp_no) ON DELETE CASCADE,
    dept_no text REFERENCES department(dept_no) ON DELETE CASCADE,
    from_date date NOT NULL,
    to_date date NOT NULL,
    PRIMARY KEY (emp_no, dept_no)
);

--
-- Name: salary; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS salary (
    emp_no integer REFERENCES employee(emp_no) ON DELETE CASCADE,
    amount integer NOT NULL,
    from_date date,
    to_date date NOT NULL,
    PRIMARY KEY (emp_no, from_date)
);

--
-- Name: idx_salary_amount; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_salary_amount ON salary (amount);

--
-- Name: title; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS title (
    emp_no integer REFERENCES employee(emp_no) ON DELETE CASCADE,
    title text,
    from_date date,
    to_date date,
    PRIMARY KEY (emp_no, title, from_date)
);

