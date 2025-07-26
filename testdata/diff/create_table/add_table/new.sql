CREATE TABLE public.departments (
    id integer PRIMARY KEY,
    name text NOT NULL
);

CREATE TABLE public.users (
    id integer PRIMARY KEY,
    name text,
    email text UNIQUE,
    department_id integer REFERENCES public.departments(id)
);