CREATE TABLE public.merge_request (
    data jsonb NOT NULL,
    iid integer NOT NULL GENERATED ALWAYS AS ((data ->> 'iid')::integer) STORED
);