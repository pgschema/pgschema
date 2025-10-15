CREATE TABLE public.documents (
    id uuid,
    title text NOT NULL,
    content text,
    CONSTRAINT documents_pkey PRIMARY KEY (id)
);