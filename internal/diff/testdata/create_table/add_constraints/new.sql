CREATE TABLE public.changesets (
    id integer NOT NULL,
    repo_id integer NOT NULL,
    batch_change_id integer,
    external_id text,
    external_service_type text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT changesets_external_id_unique UNIQUE (repo_id, external_id),
    CONSTRAINT changesets_external_service_check CHECK ((external_service_type IN ('github', 'gitlab', 'bitbucket')))
);