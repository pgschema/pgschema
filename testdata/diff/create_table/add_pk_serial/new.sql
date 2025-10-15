CREATE TABLE public.users (
    id serial,
    username text NOT NULL,
    email text,
    CONSTRAINT users_pkey PRIMARY KEY (id)
);