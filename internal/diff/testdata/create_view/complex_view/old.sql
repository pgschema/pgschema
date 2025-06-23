CREATE TABLE public.users (
    id integer NOT NULL,
    username text NOT NULL
);

CREATE TABLE public.orgs (
    id integer NOT NULL,
    name text NOT NULL
);

CREATE TABLE public.prompts (
    id integer NOT NULL,
    name text NOT NULL,
    description text,
    definition_text text,
    draft boolean DEFAULT false,
    visibility_secret boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    owner_user_id integer,
    owner_org_id integer
);