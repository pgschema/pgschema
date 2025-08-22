CREATE TABLE public.users (
    id integer NOT NULL,
    name text DEFAULT 'Unknown'::text,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);