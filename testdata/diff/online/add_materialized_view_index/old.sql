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
