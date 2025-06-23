CREATE TABLE public.lsif_data_metadata (
    id integer NOT NULL,
    upload_id integer NOT NULL,
    num_result_chunks integer
);

CREATE TABLE public.lsif_data_documents (
    id integer NOT NULL,
    upload_id integer NOT NULL,
    path text NOT NULL,
    data bytea
);

CREATE TABLE public.lsif_data_result_chunks (
    id integer NOT NULL,
    upload_id integer NOT NULL,
    idx integer NOT NULL,
    data bytea
);

CREATE TABLE public.lsif_uploads (
    id integer NOT NULL,
    commit text NOT NULL,
    root text DEFAULT ''::text NOT NULL,
    uploaded_at timestamp with time zone DEFAULT now() NOT NULL
);