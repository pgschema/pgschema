--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: employees; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS employees (
    id integer NOT NULL,
    name text NOT NULL,
    department text,
    salary numeric,
    active boolean DEFAULT true
);

--
-- Name: employee_view; Type: VIEW; Schema: -; Owner: -
--

CREATE OR REPLACE VIEW employee_view AS
 SELECT id,
    name,
    department,
    salary
   FROM employees e
  WHERE active = true;


COMMENT ON VIEW employee_view IS 'Shows all active employees';

