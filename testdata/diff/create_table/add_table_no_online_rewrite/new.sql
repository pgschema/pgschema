-- First create a referenced table for FK constraint
CREATE TABLE public.companies (
    id integer PRIMARY KEY,
    name text NOT NULL
);

-- Main table with constraints and index
CREATE TABLE public.departments (
    id integer PRIMARY KEY,
    name text NOT NULL,
    company_id integer NOT NULL REFERENCES companies(id),
    budget numeric(10,2),
    created_at timestamp DEFAULT now(),
    CHECK (budget > 0)
);

CREATE INDEX idx_departments_name ON public.departments (name);