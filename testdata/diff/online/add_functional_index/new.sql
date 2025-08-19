CREATE TABLE public.users (
    id integer NOT NULL,
    first_name text,
    last_name text,
    email text,
    phone text,
    created_at timestamp with time zone,
    status text
);

CREATE INDEX idx_users_fullname_search ON public.users (lower(first_name), lower(last_name), lower(email));