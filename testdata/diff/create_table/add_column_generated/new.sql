CREATE TABLE public.merge_request (
    data jsonb NOT NULL,
    iid integer PRIMARY KEY GENERATED ALWAYS AS ((data ->> 'iid')::integer) STORED,
    title text GENERATED ALWAYS AS (data ->> 'title') STORED
);