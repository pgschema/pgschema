CREATE TABLE public.countries (
    code text,
    name text NOT NULL,
    continent text,
    CONSTRAINT countries_pkey PRIMARY KEY (code)
);