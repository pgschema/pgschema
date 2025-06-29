CREATE TABLE public.products (
    id bigint GENERATED ALWAYS AS IDENTITY,
    name text NOT NULL,
    price numeric(10,2)
);