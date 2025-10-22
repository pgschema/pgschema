CREATE TABLE public.employees (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    email VARCHAR(100),
    bio TEXT,
    status VARCHAR(20) NOT NULL,
    department_id INTEGER,
    priority INTEGER
);

CREATE TABLE public.departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    manager_id INTEGER
);