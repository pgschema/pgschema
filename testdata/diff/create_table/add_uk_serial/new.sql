CREATE TABLE public.users (
    id serial UNIQUE,
    username text NOT NULL,
    email text
);