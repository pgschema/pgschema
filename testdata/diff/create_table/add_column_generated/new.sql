CREATE TABLE public.merge_request (
    data jsonb NOT NULL,
    iid integer GENERATED ALWAYS AS ((data ->> 'iid')::integer) STORED,
    title text GENERATED ALWAYS AS (data ->> 'title') STORED,
    CONSTRAINT pk_merge_request_iid PRIMARY KEY (iid)
);