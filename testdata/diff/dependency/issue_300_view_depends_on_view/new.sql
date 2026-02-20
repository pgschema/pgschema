-- Base table
CREATE TABLE public.priority (
    id integer PRIMARY KEY,
    name text NOT NULL,
    level integer NOT NULL
);

-- Base table
CREATE TABLE public.priority_user (
    id integer PRIMARY KEY,
    user_id integer NOT NULL,
    priority_id integer REFERENCES public.priority(id)
);

-- View that is depended upon (must be created first)
CREATE OR REPLACE VIEW public.priority_expanded AS
SELECT pu.id, pu.user_id, p.name AS priority_name, p.level
FROM public.priority_user pu
JOIN public.priority p ON p.id = pu.priority_id;

-- Base table for activity
CREATE TABLE public.activity_x (
    id integer PRIMARY KEY,
    title text NOT NULL,
    priority_user_id integer
);

-- View that depends on priority_expanded (must be created second)
-- Alphabetically "activity" comes before "priority_expanded"
CREATE OR REPLACE VIEW public.activity AS
SELECT a.id, a.title, upe.priority_name
FROM public.activity_x a
JOIN public.priority_expanded upe ON upe.id = a.priority_user_id;
