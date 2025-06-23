ALTER TABLE public.users 
ADD CONSTRAINT users_email_key UNIQUE (email);