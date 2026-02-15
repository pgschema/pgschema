--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 18.0
-- Dumped by pgschema version 1.7.0


--
-- Name: test_table; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS test_table (
    id integer,
    range_col int4range NOT NULL,
    CONSTRAINT test_table_pkey PRIMARY KEY (id),
    CONSTRAINT excl_no_overlap EXCLUDE USING gist (range_col WITH &&)
);

