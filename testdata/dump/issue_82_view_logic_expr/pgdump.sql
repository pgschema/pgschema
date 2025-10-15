--
-- PostgreSQL database dump
--

-- Dumped from database version 17.5 (Debian 17.5-1.pgdg120+1)
-- Dumped by pg_dump version 17.5 (Homebrew)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
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
-- Name: orders; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.orders (
    id integer NOT NULL,
    status character varying(50) NOT NULL,
    amount numeric(10,2)
);


--
-- Name: paid_orders; Type: VIEW; Schema: public; Owner: -
--

CREATE VIEW public.paid_orders AS
 SELECT id AS order_id,
    status,
        CASE
            WHEN ((status)::text = ANY ((ARRAY['paid'::character varying, 'completed'::character varying])::text[])) THEN amount
            ELSE NULL::numeric
        END AS paid_amount
   FROM public.orders
  ORDER BY id, status;


--
-- Name: orders orders_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.orders
    ADD CONSTRAINT orders_pkey PRIMARY KEY (id);


--
-- PostgreSQL database dump complete
--

