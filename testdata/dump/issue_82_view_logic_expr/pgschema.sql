--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.4.0


--
-- Name: orders; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS orders (
    id integer,
    status varchar(50) NOT NULL,
    amount numeric(10,2),
    CONSTRAINT orders_pkey PRIMARY KEY (id)
);

--
-- Name: paid_orders; Type: VIEW; Schema: -; Owner: -
--

CREATE OR REPLACE VIEW paid_orders AS
 SELECT id AS order_id,
    status,
        CASE
            WHEN status::text = ANY (ARRAY['paid'::character varying::text, 'completed'::character varying::text]) THEN amount
            ELSE NULL::numeric
        END AS paid_amount
   FROM orders
  ORDER BY id, status;

