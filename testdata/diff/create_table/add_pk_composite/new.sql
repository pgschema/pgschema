CREATE TABLE public.user_permissions (
    user_id integer NOT NULL,
    resource_id integer NOT NULL,
    permission_type text NOT NULL,
    granted_at timestamp with time zone,
    CONSTRAINT user_permissions_pkey PRIMARY KEY (user_id, resource_id, permission_type)
);