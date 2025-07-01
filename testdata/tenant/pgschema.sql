--
-- PostgreSQL database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 0.1.1


--
-- Name: posts_id_seq; Type: SEQUENCE; Schema: -; Owner: -
--

CREATE SEQUENCE posts_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: -; Owner: -
--

CREATE SEQUENCE users_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: posts; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE posts (
    id integer DEFAULT nextval('posts_id_seq'::regclass) NOT NULL PRIMARY KEY,
    title character varying(200) NOT NULL,
    content text,
    author_id integer,
    status public.status DEFAULT 'active'::status,
    created_at timestamp without time zone DEFAULT now()
);


--
-- Name: users; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE users (
    id integer DEFAULT nextval('users_id_seq'::regclass) NOT NULL PRIMARY KEY,
    username character varying(100) NOT NULL,
    email character varying(100) NOT NULL,
    role public.user_role DEFAULT 'user'::user_role,
    status public.status DEFAULT 'active'::status,
    created_at timestamp without time zone DEFAULT now()
);


--
-- Name: posts posts_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY posts
    ADD CONSTRAINT posts_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: idx_users_email; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_users_email ON users USING btree (email);


--
-- Name: posts posts_author_id_fkey; Type: FK CONSTRAINT; Schema: -; Owner: -
--

ALTER TABLE ONLY posts
    ADD CONSTRAINT posts_author_id_fkey FOREIGN KEY (author_id) REFERENCES users(id);


--
-- PostgreSQL database dump complete
--

