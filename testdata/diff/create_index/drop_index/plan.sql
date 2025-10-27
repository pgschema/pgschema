DROP INDEX IF EXISTS idx_products_category_price;

CREATE MATERIALIZED VIEW IF NOT EXISTS product_summary AS
SELECT
    id,
    name,
    price
FROM products;
