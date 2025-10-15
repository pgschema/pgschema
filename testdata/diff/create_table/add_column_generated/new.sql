CREATE TABLE public.merge_request (
    data jsonb NOT NULL,
    iid integer GENERATED ALWAYS AS ((data ->> 'iid')::integer) STORED,
    title text GENERATED ALWAYS AS (data ->> 'title') STORED,
    cleaned_title varchar(255) GENERATED ALWAYS AS (lower(data ->> 'title')) STORED NOT NULL,
    CONSTRAINT pk_merge_request_iid PRIMARY KEY (iid)
);