--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: status; Type: TYPE; Schema: -; Owner: -
--

CREATE TYPE status AS ENUM (
    'active',
    'inactive',
    'pending'
);

