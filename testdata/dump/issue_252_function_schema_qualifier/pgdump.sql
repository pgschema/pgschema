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
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id integer NOT NULL,
    name text NOT NULL,
    email text
);

--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.users_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;

--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);

--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

--
-- Name: get_user_count(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.get_user_count() RETURNS integer
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN (SELECT count(*)::integer FROM public.users);
END;
$$;

--
-- Name: get_user_by_name(text); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.get_user_by_name(p_name text) RETURNS TABLE(id integer, name text, email text)
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN QUERY SELECT u.id, u.name, u.email FROM public.users u WHERE u.name = p_name;
END;
$$;

--
-- Name: get_table_info(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.get_table_info() RETURNS text
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN 'Table: public.users';
END;
$$;

--
-- Name: insert_user(text, text); Type: PROCEDURE; Schema: public; Owner: -
--

CREATE PROCEDURE public.insert_user(IN p_name text, IN p_email text)
    LANGUAGE plpgsql
    AS $$
BEGIN
    INSERT INTO public.users (name, email) VALUES (p_name, p_email);
END;
$$;

--
-- PostgreSQL database dump complete
--
