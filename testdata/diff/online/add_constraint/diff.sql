ALTER TABLE orders
ADD CONSTRAINT check_amount_positive CHECK (amount > 0::numeric);
