ALTER TABLE orders
ADD CONSTRAINT check_amount_positive CHECK (amount > 0::numeric);
ALTER TABLE orders
ADD CONSTRAINT check_valid_status CHECK (status::text IN ('pending'::character varying, 'shipped'::character varying, 'delivered'::character varying));
