--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: user_rating; Type: DOMAIN; Schema: -; Owner: -
--

CREATE DOMAIN user_rating AS integer
  DEFAULT 3
  CONSTRAINT user_rating_check CHECK ((VALUE >= 1) AND (VALUE <= 10));

