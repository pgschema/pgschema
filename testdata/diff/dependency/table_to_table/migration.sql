CREATE TABLE departments (
    id integer PRIMARY KEY,
    name text NOT NULL
);

CREATE TABLE users (
    id integer PRIMARY KEY,
    name text,
    email text UNIQUE,
    department_id integer REFERENCES departments(id)
);