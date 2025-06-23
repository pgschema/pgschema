CREATE TABLE public.users (
    id integer NOT NULL,
    username text NOT NULL
);

CREATE TABLE public.batch_spec_resolution_jobs (
    id integer NOT NULL,
    batch_spec_id integer NOT NULL,
    state text NOT NULL
);