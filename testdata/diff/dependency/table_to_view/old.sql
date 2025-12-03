-- Base table referenced by a view
CREATE TABLE products (
    id integer PRIMARY KEY,
    name text NOT NULL,
    price numeric(10,2)
);

-- View that depends on the products table
CREATE VIEW expensive_products AS
SELECT id, name, price FROM products WHERE price > 100;
