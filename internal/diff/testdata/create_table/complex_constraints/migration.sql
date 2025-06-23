ALTER TABLE public.batch_changes ADD COLUMN updated_at timestamp with time zone DEFAULT now() NOT NULL;

ALTER TABLE public.batch_changes ADD COLUMN created_at timestamp with time zone DEFAULT now() NOT NULL;

ALTER TABLE public.batch_changes ALTER COLUMN name SET NOT NULL;