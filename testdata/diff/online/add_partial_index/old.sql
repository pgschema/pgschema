CREATE TYPE public.order_status AS ENUM ('pending', 'processing', 'confirmed', 'shipped', 'delivered', 'cancelled');

CREATE TABLE public.orders (
    id integer NOT NULL,
    customer_id integer,
    status order_status,
    order_date date,
    total_amount numeric(10,2),
    payment_status text,
    is_active boolean,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);