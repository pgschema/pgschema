--
-- PostgreSQL database dump
--

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: user; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public."user" (
    id integer NOT NULL,
    name text NOT NULL,
    email text
);

--
-- Name: user_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: user_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_id_seq OWNED BY public."user".id;

--
-- Name: user id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public."user" ALTER COLUMN id SET DEFAULT nextval('public.user_id_seq'::regclass);

--
-- Name: user user_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public."user"
    ADD CONSTRAINT user_pkey PRIMARY KEY (id);

--
-- Name: get_first_user(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.get_first_user() RETURNS text
    LANGUAGE plpgsql
    AS $$
DECLARE
    account public.user;
BEGIN
    SELECT * INTO account FROM public.user LIMIT 1;
    RETURN account.name;
END;
$$;

--
-- Name: count_users(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.count_users() RETURNS integer
    LANGUAGE plpgsql
    AS $$
DECLARE
    total_count integer;
BEGIN
    SELECT count(*)::integer INTO total_count FROM public."user";
    RETURN total_count;
END;
$$;

--
-- PostgreSQL database dump complete
--
