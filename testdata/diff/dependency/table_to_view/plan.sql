ALTER TABLE products ADD COLUMN category text;

CREATE OR REPLACE VIEW expensive_products AS
 SELECT id,
    name,
    price,
    category
   FROM products
  WHERE price > 100::numeric;
