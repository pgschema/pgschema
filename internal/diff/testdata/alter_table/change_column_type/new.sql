CREATE TABLE public.user_pending_permissions (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    permission text NOT NULL,
    object_ids_ints bigint[]
);