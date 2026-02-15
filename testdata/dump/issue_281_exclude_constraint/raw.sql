--
-- Test case for GitHub issue #281: EXCLUDE constraint incorrectly dumped as regular INDEX
--
-- This test verifies that EXCLUDE USING gist constraints are preserved as proper
-- table-level constraints in dump output, not converted to CREATE INDEX statements.
--
-- Uses int4range with && operator which has native GiST support (no btree_gist needed).
--

CREATE TABLE test_table (
    id integer PRIMARY KEY,
    range_col int4range NOT NULL,
    CONSTRAINT excl_no_overlap EXCLUDE USING gist (range_col WITH &&)
);
