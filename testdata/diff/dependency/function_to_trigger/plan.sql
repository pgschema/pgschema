CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name text NOT NULL,
    email text UNIQUE,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION update_modified_time()
RETURNS trigger
LANGUAGE plpgsql
SECURITY INVOKER
VOLATILE
AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE TRIGGER update_users_modified_time
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_time();