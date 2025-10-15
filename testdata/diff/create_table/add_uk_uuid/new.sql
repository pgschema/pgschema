CREATE TABLE public.documents (
    id uuid,
    title text NOT NULL,
    content text,
    CONSTRAINT documents_id_key UNIQUE (id)
);