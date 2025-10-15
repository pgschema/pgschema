CREATE TABLE public.users (
    id integer GENERATED ALWAYS AS IDENTITY,
    username text NOT NULL,
    email text,
    CONSTRAINT users_pkey PRIMARY KEY (id)
);