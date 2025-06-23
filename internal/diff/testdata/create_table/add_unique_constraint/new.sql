CREATE TABLE public.users (
    id integer NOT NULL,
    email text,
    CONSTRAINT users_email_key UNIQUE (email)
);