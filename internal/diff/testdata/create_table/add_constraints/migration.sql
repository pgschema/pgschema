ALTER TABLE public.changesets ADD COLUMN created_at timestamp with time zone DEFAULT now() NOT NULL;
ALTER TABLE public.changesets ADD COLUMN updated_at timestamp with time zone DEFAULT now() NOT NULL;

ALTER TABLE public.changesets 
ADD CONSTRAINT changesets_external_id_unique UNIQUE (repo_id, external_id);

ALTER TABLE public.changesets 
ADD CONSTRAINT changesets_external_service_check CHECK (external_service_type IN ('github', 'gitlab', 'bitbucket'));