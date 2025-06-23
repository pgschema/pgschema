CREATE TABLE public.changesets (
    id integer NOT NULL,
    repo_id integer NOT NULL,
    batch_change_id integer,
    external_id text,
    external_service_type text
);