ALTER TABLE public.permissions ALTER COLUMN id TYPE bigint;

ALTER TABLE public.permissions ALTER COLUMN user_id TYPE bigint;

ALTER TABLE public.permissions ALTER COLUMN repo_ids TYPE bigint[];