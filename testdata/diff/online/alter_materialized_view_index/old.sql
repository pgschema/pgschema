CREATE TABLE public.users (
    id integer NOT NULL,
    email text,
    username text,
    created_at timestamp with time zone,
    status text
);

CREATE MATERIALIZED VIEW public.user_summary AS
SELECT
    id,
    email,
    username,
    created_at,
    status
FROM users;

CREATE INDEX idx_user_summary_email ON public.user_summary (email);
