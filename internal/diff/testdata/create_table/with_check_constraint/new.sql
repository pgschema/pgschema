CREATE TABLE public.users (
    id integer NOT NULL,
    age integer,
    CONSTRAINT users_age_check CHECK ((age >= 0))
);