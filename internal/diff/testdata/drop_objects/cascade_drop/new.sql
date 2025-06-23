CREATE TABLE public.lsif_uploads (
    id integer NOT NULL,
    commit text NOT NULL,
    root text DEFAULT ''::text NOT NULL,
    uploaded_at timestamp with time zone DEFAULT now() NOT NULL
);