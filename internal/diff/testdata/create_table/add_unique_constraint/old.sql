CREATE TABLE public.user_sessions (
    user_id integer NOT NULL,
    session_token text NOT NULL,
    device_fingerprint text NOT NULL,
    created_at timestamp NOT NULL
);