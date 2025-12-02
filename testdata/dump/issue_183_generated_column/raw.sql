--
-- Test case for GitHub issue #183: Generated column expression handling
--
-- This demonstrates three scenarios from the bug report:
-- 1. Type casting in boolean expression
-- 2. String concatenation with TRIM function
-- 3. tsvector generation with COALESCE
--

--
-- Case 1: Boolean expression with type casting
-- Tests: ((title)::text = 'Public Data'::text)
--
CREATE TABLE snapshots (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    title text NOT NULL,
    is_live boolean GENERATED ALWAYS AS (((title)::text = 'Public Data'::text)) STORED,
    CONSTRAINT snapshots_pkey PRIMARY KEY (id)
);

--
-- Case 2: String concatenation with TRIM function
-- Tests: TRIM(BOTH FROM ((firstname || ' '::text) || lastname))
--
CREATE TABLE users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    firstname text NOT NULL,
    lastname text NOT NULL,
    full_name text GENERATED ALWAYS AS (TRIM(BOTH FROM ((firstname || ' '::text) || lastname))) STORED,
    CONSTRAINT users_pkey PRIMARY KEY (id)
);

--
-- Case 3: tsvector with COALESCE expression
-- Tests: to_tsvector('english'::regconfig, COALESCE(item, ''::text))
--
CREATE TABLE list_items (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    item text,
    fulltext tsvector GENERATED ALWAYS AS (to_tsvector('english'::regconfig, COALESCE(item, ''::text))) STORED,
    CONSTRAINT list_items_pkey PRIMARY KEY (id)
);
