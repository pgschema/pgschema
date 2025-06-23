CREATE TABLE public.permissions (
    id integer NOT NULL,
    user_id integer NOT NULL,
    repo_ids integer[]
);