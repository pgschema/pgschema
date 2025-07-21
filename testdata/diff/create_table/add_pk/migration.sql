ALTER TABLE public.order_items
ADD CONSTRAINT order_items_pkey PRIMARY KEY (order_id, product_id);