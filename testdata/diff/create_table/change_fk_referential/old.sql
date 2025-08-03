CREATE TABLE public.department (
    dept_no text NOT NULL,
    dept_name text NOT NULL,
    PRIMARY KEY (dept_no)
);

CREATE TABLE public.dept_emp (
    emp_no integer NOT NULL,
    dept_no text NOT NULL,
    from_date date NOT NULL,
    to_date date NOT NULL,
    PRIMARY KEY (emp_no, dept_no),
    FOREIGN KEY (dept_no) REFERENCES public.department (dept_no)
);