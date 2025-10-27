CREATE TABLE IF NOT EXISTS companies (
    id integer,
    name text NOT NULL,
    CONSTRAINT companies_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS departments (
    id integer,
    name text NOT NULL,
    company_id integer NOT NULL,
    budget numeric(10,2),
    created_at timestamp DEFAULT now(),
    CONSTRAINT departments_pkey PRIMARY KEY (id),
    CONSTRAINT departments_company_id_fkey FOREIGN KEY (company_id) REFERENCES companies (id),
    CONSTRAINT departments_budget_check CHECK (budget > 0)
);

CREATE INDEX IF NOT EXISTS idx_departments_name ON departments (name);
