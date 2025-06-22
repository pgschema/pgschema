CREATE TABLE public.users (
    id integer NOT NULL,
    name text
);

CREATE TABLE public.posts (
    id integer NOT NULL,
    title text,
    user_id integer
);