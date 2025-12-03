-- Base table referenced by a materialized view
CREATE TABLE products (
    id integer PRIMARY KEY,
    name text NOT NULL,
    price numeric(10,2)
);

-- Materialized view that depends on the products table
CREATE MATERIALIZED VIEW expensive_products AS
SELECT id, name, price FROM products WHERE price > 100;
