CREATE TABLE public.companies (
    tenant_id integer NOT NULL,
    company_id integer NOT NULL,
    company_name text NOT NULL,
    CONSTRAINT companies_pkey PRIMARY KEY (tenant_id, company_id)
);

CREATE TABLE public.employees (
    id integer NOT NULL,
    employee_number text NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    tenant_id integer NOT NULL,
    company_id integer NOT NULL
);