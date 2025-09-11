--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: employees; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS employees (
    id SERIAL PRIMARY KEY,
    name varchar(100) NOT NULL,
    salary numeric(10,2) NOT NULL,
    status varchar(20) DEFAULT 'active'
);

--
-- Name: active_employees; Type: VIEW; Schema: -; Owner: -
--

CREATE MATERIALIZED VIEW IF NOT EXISTS active_employees AS
 SELECT id,
    name,
    salary
   FROM employees
  WHERE status::text = 'active'::text;

