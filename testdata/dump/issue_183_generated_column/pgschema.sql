--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.4.3


--
-- Name: list_items; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS list_items (
    id uuid DEFAULT gen_random_uuid(),
    item text,
    fulltext tsvector GENERATED ALWAYS AS (to_tsvector('english'::regconfig, COALESCE(item, ''::text))) STORED,
    CONSTRAINT list_items_pkey PRIMARY KEY (id)
);

--
-- Name: snapshots; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS snapshots (
    id uuid DEFAULT gen_random_uuid(),
    title text NOT NULL,
    is_live boolean GENERATED ALWAYS AS ((title = 'Public Data'::text)) STORED,
    CONSTRAINT snapshots_pkey PRIMARY KEY (id)
);

--
-- Name: users; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS users (
    id uuid DEFAULT gen_random_uuid(),
    firstname text NOT NULL,
    lastname text NOT NULL,
    full_name text GENERATED ALWAYS AS (TRIM(BOTH FROM ((firstname || ' '::text) || lastname))) STORED,
    CONSTRAINT users_pkey PRIMARY KEY (id)
);

