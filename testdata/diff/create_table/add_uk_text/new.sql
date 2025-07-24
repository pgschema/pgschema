CREATE TABLE public.countries (
    code text UNIQUE,
    name text NOT NULL,
    continent text
);