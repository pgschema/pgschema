--
-- PostgreSQL database dump for quote-all testing
--

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

--
-- Test table with normal identifiers (should not require quoting without --quote-all)
--
CREATE TABLE public.users (
    id integer NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    email text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

--
-- Test table with reserved word identifier (should always require quoting)
--
CREATE TABLE public."order" (
    id integer NOT NULL,
    user_id integer NOT NULL,
    total_amount numeric(10,2),
    status text DEFAULT 'pending'
);

--
-- Test table with mixed case identifiers (should require quoting)
--
CREATE TABLE public."MixedCase" (
    "ID" integer NOT NULL,
    "FirstName" text,
    "LastName" text,
    "SpecialColumn" text
);

--
-- Test sequence with normal name
--
CREATE SEQUENCE public.users_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Test sequence with mixed case name
--
CREATE SEQUENCE public."OrderSeq"
    AS integer
    START WITH 1000
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Test index with normal name
--
CREATE INDEX idx_users_email ON public.users USING btree (email);

--
-- Test index with reserved word and mixed case
--
CREATE INDEX "Index_Order_Status" ON public."order" USING btree (status);

--
-- Test view with normal name
--
CREATE VIEW public.user_orders AS
 SELECT u.id,
    u.first_name,
    u.last_name,
    o.total_amount,
    o.status
   FROM (public.users u
     LEFT JOIN public."order" o ON ((u.id = o.user_id)));

--
-- Test function with normal name
--
CREATE FUNCTION public.get_user_count() RETURNS integer
    LANGUAGE sql
    AS $$
    SELECT COUNT(*) FROM users;
$$;

--
-- Add constraints
--
ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public."order"
    ADD CONSTRAINT order_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public."MixedCase"
    ADD CONSTRAINT "MixedCase_pkey" PRIMARY KEY ("ID");

--
-- Add foreign key constraint
--
ALTER TABLE ONLY public."order"
    ADD CONSTRAINT order_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);

--
-- Set sequence ownership
--
ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;
ALTER SEQUENCE public."OrderSeq" OWNED BY public."order".id;

--
-- Comments on objects
--
COMMENT ON TABLE public.users IS 'Table storing user information';
COMMENT ON COLUMN public.users.first_name IS 'User first name';
COMMENT ON TABLE public."order" IS 'Table storing order information';
COMMENT ON COLUMN public."MixedCase"."FirstName" IS 'Mixed case column comment';