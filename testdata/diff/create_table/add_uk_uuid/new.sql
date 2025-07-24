CREATE TABLE public.documents (
    id uuid UNIQUE,
    title text NOT NULL,
    content text
);