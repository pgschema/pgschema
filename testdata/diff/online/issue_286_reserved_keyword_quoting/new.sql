-- Test Case: Reserved keyword table names must be quoted in online rewrite DDL
-- Adding a column with FK and an index on a table named "order" (reserved keyword).

CREATE TABLE public.tenant (
    id uuid NOT NULL,
    name text NOT NULL,
    CONSTRAINT tenant_pkey PRIMARY KEY (id)
);

CREATE TABLE public."order" (
    id integer NOT NULL,
    order_number text NOT NULL,
    tenant_id uuid CONSTRAINT "FK_order_tenant" REFERENCES public.tenant (id),
    CONSTRAINT order_pkey PRIMARY KEY (id)
);

CREATE INDEX "IDX_order_tenant_order_number" ON public."order" (tenant_id, order_number);
