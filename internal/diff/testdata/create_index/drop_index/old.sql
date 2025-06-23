CREATE TABLE public.products (
    id integer NOT NULL,
    name text,
    price numeric(10,2),
    category_id integer
);

CREATE INDEX idx_products_category_price ON public.products USING btree (category_id, price DESC);