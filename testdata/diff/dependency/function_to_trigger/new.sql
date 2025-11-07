-- Table (unchanged)
CREATE TABLE public.users (
    id serial PRIMARY KEY,
    name text NOT NULL,
    email text UNIQUE,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);

-- Different function for logging changes
CREATE OR REPLACE FUNCTION public.log_user_changes()
RETURNS trigger AS $$
BEGIN
    RAISE NOTICE 'User record changed: %', NEW.id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Different trigger that depends on the log function
CREATE TRIGGER log_users_trigger
    AFTER INSERT OR UPDATE ON public.users
    FOR EACH ROW
    EXECUTE FUNCTION public.log_user_changes();
