-- Table structure changed (added column), view updated to use it
CREATE TABLE products (
    id integer PRIMARY KEY,
    name text NOT NULL,
    price numeric(10,2),
    category text
);

-- View now includes category
CREATE VIEW expensive_products AS
SELECT id, name, price, category FROM products WHERE price > 100;
