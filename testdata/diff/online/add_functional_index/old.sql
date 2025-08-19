CREATE TABLE public.users (
    id integer NOT NULL,
    first_name text,
    last_name text,
    email text,
    phone text,
    created_at timestamp with time zone,
    status text
);