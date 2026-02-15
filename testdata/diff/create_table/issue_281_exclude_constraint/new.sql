CREATE TABLE IF NOT EXISTS test_table (
    id integer,
    range_col int4range NOT NULL,
    CONSTRAINT test_table_pkey PRIMARY KEY (id),
    CONSTRAINT excl_no_overlap EXCLUDE USING gist (range_col WITH &&)
);
