ALTER TABLE test_table
ADD CONSTRAINT excl_no_overlap EXCLUDE USING gist (range_col WITH &&);
