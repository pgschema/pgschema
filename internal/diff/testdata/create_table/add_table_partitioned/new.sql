CREATE TABLE public.orders (
    id integer NOT NULL,
    order_date date NOT NULL,
    customer_id integer
) PARTITION BY RANGE (order_date);