--
-- Name: tenants; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tenants (
    id bigint NOT NULL,
    name text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT tenants_name_length CHECK ((( >= 3) AND ( <= 32))),
    CONSTRAINT tenants_name_valid CHECK ((name ~ ))
);

ALTER TABLE public.users ADD COLUMN tenant_id bigint;

ALTER TABLE public.posts ADD COLUMN tenant_id bigint;