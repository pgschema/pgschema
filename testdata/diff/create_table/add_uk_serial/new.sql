CREATE TABLE public.users (
    id serial,
    username text NOT NULL,
    email text,
    CONSTRAINT users_id_key UNIQUE (id)
);