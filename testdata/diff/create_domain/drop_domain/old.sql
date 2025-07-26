CREATE DOMAIN product_code AS varchar(20)
  NOT NULL
  CHECK (VALUE ~ '^[A-Z]{2}-\d{4}-[A-Z]{2}$');