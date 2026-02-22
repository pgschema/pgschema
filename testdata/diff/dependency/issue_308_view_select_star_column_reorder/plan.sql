ALTER TABLE item ADD COLUMN new_col text;

DROP VIEW IF EXISTS item_extended RESTRICT;

CREATE OR REPLACE VIEW item_extended AS
 SELECT i.id,
    i.title,
    i.status,
    i.new_col,
    c.name AS category_name
   FROM item i
     JOIN category c ON c.id = i.id;
