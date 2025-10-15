CREATE TABLE public.users (
    id integer,
    username text NOT NULL,
    email text,
    CONSTRAINT users_id_key UNIQUE (id)
);