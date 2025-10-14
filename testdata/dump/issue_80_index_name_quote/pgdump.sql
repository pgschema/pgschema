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
-- Name: products; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.products (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    price numeric(10,2),
    category text
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    email text NOT NULL,
    username text NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    status text,
    "position" integer,
    department text
);


--
-- Name: products products_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.products
    ADD CONSTRAINT products_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: UPPER name search; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX "UPPER name search" ON public.products USING btree (upper(name));


--
-- Name: UserDepartmentIndex; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX "UserDepartmentIndex" ON public.users USING btree (department);


--
-- Name: active users index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX "active users index" ON public.users USING btree (status) WHERE (status = 'active'::text);


--
-- Name: email+username combo; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX "email+username combo" ON public.users USING btree (email, username);


--
-- Name: order; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX "order" ON public.products USING btree (price DESC);


--
-- Name: products_category_idx_v2; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX products_category_idx_v2 ON public.products USING btree (category);


--
-- Name: user email index; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX "user email index" ON public.users USING btree (email);


--
-- Name: user-status-index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX "user-status-index" ON public.users USING btree (status);


--
-- Name: users.position.idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX "users.position.idx" ON public.users USING btree ("position");


--
-- Name: users_created_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX users_created_at_idx ON public.users USING btree (created_at);


--
-- PostgreSQL database dump complete
--

