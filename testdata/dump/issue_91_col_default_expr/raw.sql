--
-- Test case for GitHub issue #91: Default expression with NOT NULL dumping
--
-- This test case reproduces a bug where column default expressions containing
-- parentheses combined with NOT NULL constraints are formatted incorrectly.
--
-- The issue occurs when:
-- 1. A column has a default expression wrapped in parentheses
-- 2. The default expression contains complex SQL like `now() AT TIME ZONE 'utc'`
-- 3. The column also has a NOT NULL constraint
--
-- Original bug: DEFAULT (now() AT TIME ZONE 'utc') NOT NULL
-- Gets corrupted to: DEFAULT (now() AT TIME ZONE 'utc' NOT NULL
-- (Missing closing parenthesis before NOT NULL!)
--

--
-- Test table with complex default expressions
--
CREATE TABLE some_table (
    id serial primary key,
    created_at timestamp without time zone default (now() at time zone 'utc') not null
);
