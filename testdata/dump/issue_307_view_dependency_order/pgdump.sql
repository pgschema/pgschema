--
-- PostgreSQL database dump
--

-- Dumped from database version 17.5 (Debian 17.5-1.pgdg120+1)
-- Dumped by pg_dump version 17.5 (Homebrew)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: base_data; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.base_data (
    id integer NOT NULL,
    value text NOT NULL,
    category text
);


--
-- Name: item_summary; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.item_summary AS
 SELECT id,
    value,
    category
   FROM public.base_data
  WHERE (category IS NOT NULL);


--
-- Name: dashboard; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.dashboard AS
 SELECT id,
    value
   FROM public.item_summary;


--
-- Name: base_data base_data_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.base_data
    ADD CONSTRAINT base_data_pkey PRIMARY KEY (id);


--
-- PostgreSQL database dump complete
--

