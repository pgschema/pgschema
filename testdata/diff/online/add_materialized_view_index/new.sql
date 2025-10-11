CREATE TABLE public.users (
    id integer NOT NULL,
    email text,
    created_at timestamp with time zone
);

CREATE MATERIALIZED VIEW public.user_summary AS
SELECT
    id,
    email,
    created_at
FROM users;

CREATE INDEX idx_user_summary_created_at ON public.user_summary(created_at);
