-- Create a new table with a simple index
CREATE TABLE public.users (
    id INTEGER PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(100)
);

CREATE INDEX idx_users_name ON public.users (name);
CREATE INDEX idx_users_email ON public.users (email varchar_pattern_ops);
CREATE INDEX idx_users_id ON public.users (id);
