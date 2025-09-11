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
    name text NOT NULL,
    salary numeric(10,2),
    last_modified timestamp DEFAULT CURRENT_TIMESTAMP
);

--
-- Name: employees_update_check; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER employees_update_check
    BEFORE UPDATE ON employees
    FOR EACH ROW
    EXECUTE FUNCTION pg_catalog.suppress_redundant_updates_trigger();

