-- Test Case: Adding foreign keys with NOT VALID (online migration)
-- Scenario 1: FK referencing existing constraint (z_companies_pkey)
-- Scenario 2: FK referencing newly-added constraint (tests cross-table constraint ordering bug #248)
--
-- Table names are intentionally chosen to test cross-table constraint ordering:
-- - z_companies (referenced table) comes AFTER a_employees alphabetically
-- - Without proper topological sorting, a_employees FK would be added before z_companies UNIQUE constraint
-- - This reproduces bug #248: "ERROR: there is no unique constraint matching given keys for referenced table"

CREATE TABLE public.z_companies (
    tenant_id integer NOT NULL,
    company_id integer NOT NULL,
    company_name text NOT NULL,
    CONSTRAINT z_companies_pkey PRIMARY KEY (tenant_id, company_id)
);

CREATE TABLE public.a_employees (
    id integer NOT NULL,
    employee_number text NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    tenant_id integer NOT NULL,
    company_id integer NOT NULL,
    company_name text NOT NULL
);