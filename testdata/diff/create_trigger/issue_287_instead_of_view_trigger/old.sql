CREATE TABLE public.users (
    id serial PRIMARY KEY,
    email text NOT NULL
);

CREATE VIEW public.user_emails AS
SELECT id, email
FROM public.users;

CREATE OR REPLACE FUNCTION public.insert_user_emails()
RETURNS trigger AS $$
BEGIN
    INSERT INTO public.users (email)
    VALUES (NEW.email)
    RETURNING id, email INTO NEW.id, NEW.email;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
