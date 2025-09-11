--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: big_seq; Type: SEQUENCE; Schema: -; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS big_seq AS bigint MAXVALUE 1000000 CACHE 10;

--
-- Name: int_seq; Type: SEQUENCE; Schema: -; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS int_seq AS integer START WITH 100 CACHE 5;

--
-- Name: order_seq; Type: SEQUENCE; Schema: -; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS order_seq INCREMENT BY 10 CYCLE;

--
-- Name: small_seq; Type: SEQUENCE; Schema: -; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS small_seq AS smallint CACHE 20;

--
-- Name: user_id_seq; Type: SEQUENCE; Schema: -; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS user_id_seq;

