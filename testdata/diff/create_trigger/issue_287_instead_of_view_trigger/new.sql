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

CREATE TRIGGER trg_user_emails_insert
    INSTEAD OF INSERT ON public.user_emails
    FOR EACH ROW
    EXECUTE FUNCTION public.insert_user_emails();
