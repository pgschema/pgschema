-- Test Case: Reserved keyword table names must be quoted in online rewrite DDL
-- The table "order" is a PostgreSQL reserved keyword and must be quoted everywhere.

CREATE TABLE public.tenant (
    id uuid NOT NULL,
    name text NOT NULL,
    CONSTRAINT tenant_pkey PRIMARY KEY (id)
);

CREATE TABLE public."order" (
    id integer NOT NULL,
    order_number text NOT NULL,
    CONSTRAINT order_pkey PRIMARY KEY (id)
);
