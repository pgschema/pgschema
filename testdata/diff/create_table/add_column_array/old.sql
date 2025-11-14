CREATE TYPE status AS ENUM (
    'pending',
    'active',
    'inactive',
    'archived'
);

CREATE TABLE public.articles (
    id integer NOT NULL,
    title text
);