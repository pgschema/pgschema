CREATE TABLE public.users (
    id integer NOT NULL,
    email text,
    username text,
    created_at timestamp with time zone,
    status text
);

CREATE INDEX CONCURRENTLY idx_users_email_status ON public.users USING btree (email, status) WHERE status = 'active';