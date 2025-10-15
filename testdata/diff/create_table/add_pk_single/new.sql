CREATE TABLE public.users (
    id integer,
    username text NOT NULL,
    email text,
    CONSTRAINT users_pkey PRIMARY KEY (id)
);