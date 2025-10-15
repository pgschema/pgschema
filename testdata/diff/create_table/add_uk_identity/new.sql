CREATE TABLE public.users (
    id integer GENERATED ALWAYS AS IDENTITY,
    username text NOT NULL,
    email text,
    CONSTRAINT users_id_key UNIQUE (id)
);