ALTER TABLE public.users ALTER COLUMN name DROP NOT NULL;
ALTER TABLE public.users ALTER COLUMN email SET NOT NULL;