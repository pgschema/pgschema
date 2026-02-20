-- Table whose composite type is used as a function parameter
CREATE TABLE public.activity (
    id uuid PRIMARY KEY,
    author_id uuid
);

-- Table used by the view
CREATE TABLE public.contact (
    id uuid PRIMARY KEY,
    name text NOT NULL
);

-- View referenced in the function's return type
CREATE OR REPLACE VIEW public.actor AS
SELECT id, name FROM public.contact;

-- Function that uses the table composite type as a parameter
-- and references a view in its return type.
-- This function must be created AFTER:
--   1. The activity table (for the composite type parameter)
--   2. The actor view (for RETURNS SETOF actor)
CREATE OR REPLACE FUNCTION public.get_actor(activity activity)
    RETURNS SETOF actor ROWS 1
    LANGUAGE sql STABLE
    AS $$ SELECT actor.* FROM actor WHERE actor.id = activity.author_id $$;
