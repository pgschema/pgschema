--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: companies; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS companies (
    tenant_id integer,
    company_id integer,
    company_name text NOT NULL,
    PRIMARY KEY (tenant_id, company_id)
);

--
-- Name: employees; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS employees (
    id integer NOT NULL,
    employee_number text NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    tenant_id integer NOT NULL,
    company_id integer NOT NULL,
    FOREIGN KEY (tenant_id, company_id) REFERENCES companies (tenant_id, company_id) ON DELETE CASCADE ON UPDATE CASCADE
);

