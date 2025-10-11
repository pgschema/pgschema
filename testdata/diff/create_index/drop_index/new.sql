CREATE TABLE public.products (
    id integer NOT NULL,
    name text,
    price numeric(10,2),
    category_id integer
);

CREATE MATERIALIZED VIEW public.product_summary AS
SELECT
    id,
    name,
    price
FROM products;