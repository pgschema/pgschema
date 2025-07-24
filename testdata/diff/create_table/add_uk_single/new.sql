CREATE TABLE public.users (
    id integer UNIQUE,
    username text NOT NULL,
    email text
);