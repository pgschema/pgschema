CREATE TABLE public.employees (
    id serial PRIMARY KEY,
    name text NOT NULL,
    salary numeric(10,2),
    last_modified timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION public.update_last_modified()
RETURNS trigger AS $$
BEGIN
    NEW.last_modified = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE VIEW public.employee_emails AS
SELECT id, name
FROM public.employees;

CREATE OR REPLACE FUNCTION public.insert_employee_emails()
RETURNS trigger AS $$
BEGIN
    INSERT INTO public.employees (name)
    VALUES (NEW.name)
    RETURNING id, name INTO NEW.id, NEW.name;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;