CREATE TABLE public.users (
    id integer GENERATED ALWAYS AS IDENTITY UNIQUE,
    username text NOT NULL,
    email text
);