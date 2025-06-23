CREATE TABLE public.users (
    id integer NOT NULL,
    name text NOT NULL,
    email text NOT NULL
);

CREATE TABLE public.posts (
    id integer NOT NULL,
    title text NOT NULL,
    content text,
    user_id integer NOT NULL
);