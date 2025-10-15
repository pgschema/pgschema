--
-- Test case for GitHub issue #82: View logical expressions dumping
--
-- This test case reproduces a bug where view definitions with CASE statements
-- containing IN clauses become syntactically incorrect when dumped.
--
-- The issue occurs when:
-- 1. A view has a CASE statement with an IN clause
-- 2. The view uses column aliases
-- 3. The view has an ORDER BY clause referencing the aliased columns
--
-- Original bug: CASE WHEN status IN ('paid', 'completed') THEN amount ELSE NULL END
-- Gets corrupted to: CASE WHEN status::text = ::text THEN amount ELSE NULL END
-- (The IN clause values disappear!)
--

--
-- Base table: orders with status tracking
--
CREATE TABLE orders (
    id integer PRIMARY KEY,
    status varchar(50) NOT NULL,
    amount numeric(10,2)
);

--
-- Problematic view: Uses CASE with IN clause + column alias in ORDER BY
-- This specific combination triggers the bug
--
CREATE VIEW paid_orders AS
  SELECT
    id AS order_id,
    status,
    CASE WHEN status IN ('paid', 'completed') THEN amount ELSE NULL END AS paid_amount
  FROM orders
  ORDER BY order_id, status;
