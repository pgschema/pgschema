CREATE TABLE IF NOT EXISTS departments (
    id integer PRIMARY KEY,
    name text NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    id integer PRIMARY KEY,
    name text,
    email text UNIQUE,
    department_id integer REFERENCES departments(id)
);