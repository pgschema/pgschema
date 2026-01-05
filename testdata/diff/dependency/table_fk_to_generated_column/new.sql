-- Function used in generated column
CREATE FUNCTION public.calc_priority() RETURNS integer
    LANGUAGE sql
    IMMUTABLE
    AS $$SELECT 1$$;

-- Table with generated column using the function
CREATE TABLE public.article (
    id integer PRIMARY KEY,
    title text NOT NULL,
    priority integer GENERATED ALWAYS AS (calc_priority()) STORED
);

-- Table with FK referencing article
CREATE TABLE public.articlesource (
    id integer PRIMARY KEY,
    article_id integer NOT NULL,
    source_url text,
    CONSTRAINT articlesource_article_id_fkey FOREIGN KEY (article_id) REFERENCES public.article(id)
);
