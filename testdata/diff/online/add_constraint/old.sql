CREATE TABLE public.orders (
    id integer NOT NULL,
    customer_id integer NOT NULL,
    amount numeric(10,2),
    created_at timestamp with time zone DEFAULT now() NOT NULL
);