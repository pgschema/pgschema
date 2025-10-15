CREATE TABLE IF NOT EXISTS users (
    id SERIAL,
    name text,
    email text,
    CONSTRAINT users_pkey PRIMARY KEY (id)
);
