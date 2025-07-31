-- Table with a trigger that depends on the function
CREATE TABLE public.users (
    id serial PRIMARY KEY,
    name text NOT NULL,
    email text UNIQUE,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);

-- Trigger that depends on the function
CREATE TRIGGER update_users_modified_time
    BEFORE UPDATE ON public.users
    FOR EACH ROW
    EXECUTE FUNCTION public.update_modified_time();

-- Function that will be used by the trigger
CREATE OR REPLACE FUNCTION public.update_modified_time()
RETURNS trigger AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;