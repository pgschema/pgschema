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

CREATE INDEX idx_active_orders_customer_date ON public.orders USING btree (customer_id, order_date DESC, total_amount) WHERE (status IN ('pending'::order_status, 'processing'::order_status, 'confirmed'::order_status)) AND (is_active IS NOT NULL);