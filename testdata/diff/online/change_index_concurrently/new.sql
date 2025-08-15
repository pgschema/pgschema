CREATE TABLE public.users (
    id integer NOT NULL,
    email text,
    username text,
    created_at timestamp with time zone,
    status text,
    department text
);

CREATE INDEX CONCURRENTLY idx_users_email ON public.users (email, status);

CREATE INDEX CONCURRENTLY idx_users_status ON public.users (status, department);