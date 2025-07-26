CREATE TABLE public.organizations (
    tenant_id integer NOT NULL,
    org_id integer NOT NULL,
    org_name text NOT NULL,
    org_type text NOT NULL,
    PRIMARY KEY (tenant_id, org_id),
    UNIQUE (tenant_id, org_name)
);

CREATE TABLE public.projects (
    tenant_id integer NOT NULL,
    org_id integer NOT NULL,
    project_id integer NOT NULL,
    project_name text NOT NULL,
    project_code text NOT NULL,
    description text,
    PRIMARY KEY (tenant_id, org_id, project_id),
    FOREIGN KEY (tenant_id, org_id) REFERENCES public.organizations(tenant_id, org_id),
    UNIQUE (tenant_id, org_id, project_code)
);