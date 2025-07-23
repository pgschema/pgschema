CREATE TABLE public.employees (
    id serial PRIMARY KEY,
    name text NOT NULL,
    salary numeric(10,2),
    last_modified timestamp DEFAULT CURRENT_TIMESTAMP
);