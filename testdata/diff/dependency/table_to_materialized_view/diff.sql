DROP MATERIALIZED VIEW expensive_products RESTRICT;

ALTER TABLE products ADD COLUMN category text;

CREATE MATERIALIZED VIEW IF NOT EXISTS expensive_products AS
 SELECT id,
    name,
    price,
    category
   FROM products
  WHERE price > 100::numeric;
