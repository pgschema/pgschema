--
-- Test case for GitHub issue #80: Index name quoting
--
-- This demonstrates various scenarios where index names need quoting:
-- 1. Index names with spaces
-- 2. Index names with special characters (hyphens, dots, etc.)
-- 3. Index names with mixed case that requires quoting
-- 4. Normal index names that don't require quoting (control case)
--

--
-- Test table for index quoting scenarios
--
CREATE TABLE users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    email text NOT NULL,
    username text NOT NULL,
    created_at timestamp DEFAULT now(),
    status text,
    position integer,
    department text,
    CONSTRAINT users_pkey PRIMARY KEY (id)
);

--
-- Case 1: Index name with spaces (from the original issue)
--
CREATE UNIQUE INDEX "user email index" ON users (email);

--
-- Case 2: Index name with hyphens (special character)
--
CREATE INDEX "user-status-index" ON users (status);

--
-- Case 3: Index name with dots
--
CREATE INDEX "users.position.idx" ON users (position);

--
-- Case 4: Mixed case requiring quotes
--
CREATE INDEX "UserDepartmentIndex" ON users (department);

--
-- Case 5: Normal index name (no quotes needed)
--
CREATE INDEX users_created_at_idx ON users (created_at);

--
-- Case 6: Partial index with quoted name and spaces
--
CREATE INDEX "active users index" ON users (status) WHERE status = 'active';

--
-- Case 7: Multi-column index with special name
--
CREATE UNIQUE INDEX "email+username combo" ON users (email, username);

--
-- Additional test table for more complex scenarios
--
CREATE TABLE products (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    price numeric(10,2),
    category text,
    CONSTRAINT products_pkey PRIMARY KEY (id)
);

--
-- Case 8: Functional index with quoted name
--
CREATE INDEX "UPPER name search" ON products (upper(name));

--
-- Case 9: Index with numbers and underscores (doesn't need quotes)
--
CREATE INDEX products_category_idx_v2 ON products (category);

--
-- Case 10: Index name that looks like a keyword (needs quotes)
--
CREATE INDEX "order" ON products (price DESC);