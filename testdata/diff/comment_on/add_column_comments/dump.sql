--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: products; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    name varchar(200) NOT NULL,
    price numeric(10,2) NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP
);


COMMENT ON COLUMN products.id IS 'Unique product identifier';


COMMENT ON COLUMN products.name IS 'Product display name';


COMMENT ON COLUMN products.price IS 'Product price in USD';


COMMENT ON COLUMN products.created_at IS 'Timestamp when product was added';

