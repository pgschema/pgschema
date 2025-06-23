CREATE TABLE public.order_items (
    order_id integer NOT NULL,
    product_id integer NOT NULL,
    quantity integer NOT NULL,
    unit_price decimal(10,2) NOT NULL,
    CONSTRAINT order_items_pkey PRIMARY KEY (order_id, product_id)
);