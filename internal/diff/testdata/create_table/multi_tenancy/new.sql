CREATE TABLE public.tenants (
    id bigint NOT NULL,
    name text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT tenants_name_length CHECK ((char_length(name) >= 3) AND (char_length(name) <= 32)),
    CONSTRAINT tenants_name_valid CHECK ((name ~ '^[a-z](?:[a-z0-9\\_-])*[a-z0-9]$'::text))
);

CREATE TABLE public.users (
    id integer NOT NULL,
    name text NOT NULL,
    email text NOT NULL,
    tenant_id bigint REFERENCES public.tenants(id)
);

CREATE TABLE public.posts (
    id integer NOT NULL,
    title text NOT NULL,
    content text,
    user_id integer NOT NULL,
    tenant_id bigint REFERENCES public.tenants(id)
);