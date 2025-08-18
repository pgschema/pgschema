ALTER TABLE orders
ADD CONSTRAINT check_amount_positive CHECK (amount > 0) NOT VALID;

ALTER TABLE orders VALIDATE CONSTRAINT check_amount_positive;
