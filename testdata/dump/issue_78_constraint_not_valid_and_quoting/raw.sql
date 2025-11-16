--
-- Test case for GitHub issue #78: NOT VALID constraint dumping
--
-- This demonstrates two scenarios:
-- 1. Constraint on table WITHOUT data (validated immediately, no NOT VALID)
-- 2. Constraint on table WITH data (added with NOT VALID for non-blocking migration)
--

--
-- Case 1: Table WITHOUT data - constraint is validated immediately
--
CREATE TABLE products (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    price numeric(10,2) NOT NULL
);

-- Add constraint to empty table - no NOT VALID needed
-- pg_dump will output this as a normal constraint
ALTER TABLE products
    ADD CONSTRAINT products_price_positive CHECK (price > 0);

-- Add constraint with whitespace in name to test proper quoting
-- Tests GitHub issue #78 comment about constraint names with spaces
ALTER TABLE products
    ADD CONSTRAINT "price not negative" CHECK (price >= 0);


--
-- Case 2: Table WITH data - constraint added with NOT VALID
--
CREATE TABLE users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    email text NOT NULL
);

-- Insert data FIRST to simulate production scenario
INSERT INTO users (email) VALUES ('test@example.com');
INSERT INTO users (email) VALUES ('user@domain.com');

-- Add constraint to table with existing data
-- Uses NOT VALID to avoid blocking - typical production pattern
ALTER TABLE users
    ADD CONSTRAINT users_email_is_lower CHECK (lower(email) = email) NOT VALID;
