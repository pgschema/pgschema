CREATE TYPE public.tag_type AS ENUM ('featured', 'sale', 'new', 'popular');

CREATE TABLE public.products (
    id integer NOT NULL,
    name text NOT NULL,
    tags public.tag_type[]
);
