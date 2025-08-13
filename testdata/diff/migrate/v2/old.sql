--
-- Name: department; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE department (
    dept_no text NOT NULL,
    dept_name text NOT NULL,
    PRIMARY KEY (dept_no)
);


--
-- Name: employee; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE employee (
    emp_no SERIAL NOT NULL,
    birth_date date NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    gender text NOT NULL,
    hire_date date NOT NULL,
    PRIMARY KEY (emp_no)
);


--
-- Name: dept_emp; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE dept_emp (
    emp_no integer NOT NULL,
    dept_no text NOT NULL,
    from_date date NOT NULL,
    to_date date NOT NULL,
    PRIMARY KEY (emp_no, dept_no),
    FOREIGN KEY (dept_no) REFERENCES department (dept_no),
    FOREIGN KEY (emp_no) REFERENCES employee (emp_no)
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
    FOREIGN KEY (dept_no) REFERENCES department (dept_no),
    FOREIGN KEY (emp_no) REFERENCES employee (emp_no)
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
    FOREIGN KEY (emp_no) REFERENCES employee (emp_no)
);


--
-- Name: title; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE title (
    emp_no integer NOT NULL,
    title text NOT NULL,
    from_date date NOT NULL,
    to_date date,
    PRIMARY KEY (emp_no, title, from_date),
    FOREIGN KEY (emp_no) REFERENCES employee (emp_no)
);
