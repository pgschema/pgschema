CREATE TABLE public.departments (
    id integer PRIMARY KEY,
    name text NOT NULL,
    manager_id integer
);

CREATE TABLE public.users (
    id integer PRIMARY KEY,
    name text,
    email text UNIQUE,
    department_id integer
);

ALTER TABLE public.departments
ADD CONSTRAINT departments_manager_id_fkey FOREIGN KEY (manager_id) REFERENCES public.users(id);

ALTER TABLE public.users
ADD CONSTRAINT users_department_id_fkey FOREIGN KEY (department_id) REFERENCES public.departments(id);