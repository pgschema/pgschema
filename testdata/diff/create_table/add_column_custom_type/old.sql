-- Initial state: Basic table without custom types
CREATE TABLE public.users (
    id bigint PRIMARY KEY,
    username text NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP
);
