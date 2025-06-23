CREATE TABLE public.users (
    id integer NOT NULL,
    username text NOT NULL UNIQUE,
    email text NOT NULL UNIQUE
);