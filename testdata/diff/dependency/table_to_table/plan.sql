CREATE TABLE IF NOT EXISTS departments (
    id integer,
    name text NOT NULL,
    CONSTRAINT departments_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS users (
    id integer,
    name text,
    email text,
    department_id integer,
    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT users_email_key UNIQUE (email),
    CONSTRAINT users_department_id_fkey FOREIGN KEY (department_id) REFERENCES departments (id)
);
