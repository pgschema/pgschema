--
-- Test case for GitHub issue #307: View dependency ordering
--
-- This test reproduces a bug where views are emitted in alphabetical order
-- instead of dependency order. "dashboard" sorts before "item_summary"
-- alphabetically, but dashboard depends on item_summary and must come after it.
--

-- Base table
CREATE TABLE base_data (
    id integer PRIMARY KEY,
    value text NOT NULL,
    category text
);

-- View that other views depend on (must be created first)
CREATE VIEW item_summary AS
SELECT id, value, category
FROM base_data
WHERE category IS NOT NULL;

-- View that depends on item_summary (must be created second)
-- Alphabetically "dashboard" comes before "item_summary"
CREATE VIEW dashboard AS
SELECT id, value
FROM item_summary;
