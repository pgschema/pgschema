--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: organizations; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS organizations (
    tenant_id integer,
    org_id integer,
    org_name text NOT NULL,
    org_type text NOT NULL,
    PRIMARY KEY (tenant_id, org_id),
    UNIQUE (tenant_id, org_name)
);

--
-- Name: projects; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS projects (
    tenant_id integer,
    org_id integer,
    project_id integer,
    project_name text NOT NULL,
    project_code text NOT NULL,
    description text,
    PRIMARY KEY (tenant_id, org_id, project_id),
    UNIQUE (tenant_id, org_id, project_code),
    FOREIGN KEY (tenant_id, org_id) REFERENCES organizations (tenant_id, org_id)
);

