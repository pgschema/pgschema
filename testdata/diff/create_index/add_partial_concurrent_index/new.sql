CREATE TABLE public.orders (
    id integer NOT NULL,
    customer_id integer,
    status text,
    order_date date,
    total_amount numeric(10,2),
    payment_status text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE INDEX CONCURRENTLY idx_active_orders_customer_date ON public.orders USING btree (customer_id, order_date DESC, total_amount) WHERE status IN ('pending', 'processing', 'confirmed');