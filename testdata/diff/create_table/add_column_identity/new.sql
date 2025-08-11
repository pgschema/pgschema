CREATE TABLE public.products (
    name text NOT NULL,
    price numeric(10,2),
    id bigint GENERATED ALWAYS AS IDENTITY
);