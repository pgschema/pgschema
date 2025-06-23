CREATE TABLE public.user_profiles (
    id integer NOT NULL,
    user_id integer,
    email text,
    username text,
    organization_id integer,
    created_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE UNIQUE INDEX CONCURRENTLY idx_unique_email_org ON public.user_profiles USING btree (email, organization_id) WHERE deleted_at IS NULL;