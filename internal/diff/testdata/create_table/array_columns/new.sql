CREATE TABLE public.permissions (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    repo_ids bigint[]
);