ALTER TABLE orders
ADD CONSTRAINT check_amount CHECK (amount > 0) NOT VALID;

ALTER TABLE orders VALIDATE CONSTRAINT check_amount;
