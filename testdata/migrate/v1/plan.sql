CREATE TABLE department (
    dept_no text PRIMARY KEY,
    dept_name text NOT NULL
);

CREATE TABLE employee (
    emp_no SERIAL PRIMARY KEY,
    birth_date date NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    gender text NOT NULL,
    hire_date date NOT NULL
);

CREATE TABLE dept_emp (
    emp_no integer REFERENCES employee(emp_no),
    dept_no text REFERENCES department(dept_no),
    from_date date NOT NULL,
    to_date date NOT NULL,
    PRIMARY KEY (emp_no, dept_no)
);

CREATE TABLE dept_manager (
    emp_no integer REFERENCES employee(emp_no),
    dept_no text REFERENCES department(dept_no),
    from_date date NOT NULL,
    to_date date NOT NULL,
    PRIMARY KEY (emp_no, dept_no)
);

CREATE TABLE salary (
    emp_no integer REFERENCES employee(emp_no),
    amount integer NOT NULL,
    from_date date,
    to_date date NOT NULL,
    PRIMARY KEY (emp_no, from_date)
);

CREATE TABLE title (
    emp_no integer REFERENCES employee(emp_no),
    title text,
    from_date date,
    to_date date,
    PRIMARY KEY (emp_no, title, from_date)
);
