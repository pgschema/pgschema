--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.3.0


--
-- Name: products; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS products (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    price numeric(10,2) NOT NULL,
    CONSTRAINT products_price_positive CHECK (price > 0)
);

--
-- Name: users; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    email text NOT NULL,
    CONSTRAINT users_email_is_lower CHECK (lower(email) = email) NOT VALID
);

