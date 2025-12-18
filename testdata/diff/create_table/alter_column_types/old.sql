CREATE TABLE public.user_pending_permissions (
    id integer NOT NULL,
    user_id integer NOT NULL,
    permission text NOT NULL,
    object_ids_ints integer[],
    action text,
    status text DEFAULT 'pending'
);