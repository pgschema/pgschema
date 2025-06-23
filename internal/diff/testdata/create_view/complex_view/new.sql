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

CREATE VIEW public.prompts_view AS 
 SELECT p.id,
    p.name,
    p.description,
    p.definition_text,
    p.draft,
    p.visibility_secret,
    p.created_at,
    p.updated_at,
    COALESCE(u.username, o.name) AS owner_name
   FROM ((public.prompts p
     LEFT JOIN public.users u ON ((p.owner_user_id = u.id)))
     LEFT JOIN public.orgs o ON ((p.owner_org_id = o.id)));