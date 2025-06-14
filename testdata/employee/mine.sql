--
-- PostgreSQL database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 0.0.1


--
-- Name: log_dml_operations(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.log_dml_operations() RETURNS trigger
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
-- Name: audit; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.audit (
    id integer NOT NULL,
    operation text NOT NULL,
    query text,
    user_name text NOT NULL,
    changed_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

--
-- Name: audit_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.audit_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: audit_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.audit_id_seq OWNED BY public.audit.id;

--
-- Name: audit id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit ALTER COLUMN id SET DEFAULT nextval('public.audit_id_seq'::regclass);

--
-- Name: audit audit_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit
    ADD CONSTRAINT audit_pkey PRIMARY KEY (id);

--
-- Name: department; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.department (
    dept_no text NOT NULL,
    dept_name text NOT NULL
);

--
-- Name: department department_dept_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.department
    ADD CONSTRAINT department_dept_name_key UNIQUE (dept_name);

--
-- Name: department department_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.department
    ADD CONSTRAINT department_pkey PRIMARY KEY (dept_no);

--
-- Name: dept_emp; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.dept_emp (
    emp_no integer NOT NULL,
    dept_no text NOT NULL,
    from_date date NOT NULL,
    to_date date NOT NULL
);

--
-- Name: dept_emp dept_emp_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dept_emp
    ADD CONSTRAINT dept_emp_pkey PRIMARY KEY (emp_no, dept_no);

--
-- Name: dept_manager; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.dept_manager (
    emp_no integer NOT NULL,
    dept_no text NOT NULL,
    from_date date NOT NULL,
    to_date date NOT NULL
);

--
-- Name: dept_manager dept_manager_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dept_manager
    ADD CONSTRAINT dept_manager_pkey PRIMARY KEY (emp_no, dept_no);

--
-- Name: employee; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.employee (
    emp_no integer NOT NULL,
    birth_date date NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    gender text NOT NULL,
    hire_date date NOT NULL
);

--
-- Name: employee_emp_no_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.employee_emp_no_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: employee_emp_no_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.employee_emp_no_seq OWNED BY public.employee.emp_no;

--
-- Name: employee emp_no; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.employee ALTER COLUMN emp_no SET DEFAULT nextval('public.employee_emp_no_seq'::regclass);

--
-- Name: employee employee_gender_check; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.employee
    ADD CONSTRAINT employee_gender_check CHECK ((gender = ANY (ARRAY['M'::text, 'F'::text])));

--
-- Name: employee employee_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.employee
    ADD CONSTRAINT employee_pkey PRIMARY KEY (emp_no);

--
-- Name: salary; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.salary (
    emp_no integer NOT NULL,
    amount integer NOT NULL,
    from_date date NOT NULL,
    to_date date NOT NULL
);

--
-- Name: salary salary_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.salary
    ADD CONSTRAINT salary_pkey PRIMARY KEY (emp_no, from_date);

--
-- Name: title; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.title (
    emp_no integer NOT NULL,
    title text NOT NULL,
    from_date date NOT NULL,
    to_date date
);

--
-- Name: title title_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.title
    ADD CONSTRAINT title_pkey PRIMARY KEY (emp_no, title, from_date);

--
-- Name: current_dept_emp; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.current_dept_emp AS
 SELECT l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM (dept_emp d
     JOIN dept_emp_latest_date l ON (((d.emp_no = l.emp_no) AND (d.from_date = l.from_date) AND (l.to_date = d.to_date))));;

--
-- Name: dept_emp_latest_date; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.dept_emp_latest_date AS
 SELECT emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM dept_emp
  GROUP BY emp_no;;

--
-- Name: idx_audit_changed_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_changed_at ON public.audit USING btree (changed_at);

--
-- Name: idx_audit_operation; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_operation ON public.audit USING btree (operation);

--
-- Name: idx_audit_username; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_username ON public.audit USING btree (user_name);

--
-- Name: idx_employee_hire_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_employee_hire_date ON public.employee USING btree (hire_date);

--
-- Name: idx_salary_amount; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_salary_amount ON public.salary USING btree (amount);

--
-- Name: salary salary_log_trigger; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER salary_log_trigger AFTER DELETE OR UPDATE ON public.salary FOR EACH ROW EXECUTE FUNCTION public.log_dml_operations();

--
-- Name: dept_emp dept_emp_dept_no_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dept_emp
    ADD CONSTRAINT dept_emp_dept_no_fkey FOREIGN KEY (dept_no) REFERENCES public.department(dept_no) ON DELETE CASCADE;

--
-- Name: dept_emp dept_emp_emp_no_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dept_emp
    ADD CONSTRAINT dept_emp_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES public.employee(emp_no) ON DELETE CASCADE;

--
-- Name: dept_manager dept_manager_dept_no_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dept_manager
    ADD CONSTRAINT dept_manager_dept_no_fkey FOREIGN KEY (dept_no) REFERENCES public.department(dept_no) ON DELETE CASCADE;

--
-- Name: dept_manager dept_manager_emp_no_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dept_manager
    ADD CONSTRAINT dept_manager_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES public.employee(emp_no) ON DELETE CASCADE;

--
-- Name: salary salary_emp_no_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.salary
    ADD CONSTRAINT salary_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES public.employee(emp_no) ON DELETE CASCADE;

--
-- Name: title title_emp_no_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.title
    ADD CONSTRAINT title_emp_no_fkey FOREIGN KEY (emp_no) REFERENCES public.employee(emp_no) ON DELETE CASCADE;

--
-- Name: audit; Type: ROW SECURITY; Schema: public; Owner: -
--

ALTER TABLE public.audit ENABLE ROW LEVEL SECURITY;

--
-- Name: audit audit_insert_system; Type: POLICY; Schema: public; Owner: -
--

CREATE POLICY audit_insert_system ON public.audit FOR INSERT WITH CHECK (true);

--
-- Name: audit audit_user_isolation; Type: POLICY; Schema: public; Owner: -
--

CREATE POLICY audit_user_isolation ON public.audit USING ((user_name = CURRENT_USER));

--
-- PostgreSQL database dump complete
--

