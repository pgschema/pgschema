CREATE TABLE item (
    id uuid PRIMARY KEY,
    title text,
    status text,
    new_col text
);

CREATE TABLE category (
    id uuid PRIMARY KEY,
    name text
);

CREATE VIEW item_extended AS
SELECT i.*, c.name AS category_name
FROM item i
JOIN category c ON c.id = i.id;
