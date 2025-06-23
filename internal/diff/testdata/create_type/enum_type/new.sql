CREATE TYPE public.batch_changes_state AS ENUM (
    'DRAFT',
    'OPEN',
    'COMPLETE',
    'CLOSED'
);

CREATE TABLE public.batch_changes (
    id integer NOT NULL,
    name text NOT NULL,
    state public.batch_changes_state NOT NULL
);