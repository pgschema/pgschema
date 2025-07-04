--
-- PostgreSQL database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 0.1.4


--
-- Name: users; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE users (
    id SERIAL NOT NULL,
    username character varying(100) NOT NULL,
    email character varying(100) NOT NULL,
    role public.user_role DEFAULT 'user'::user_role,
    status public.status DEFAULT 'active'::status,
    created_at timestamp DEFAULT now(),
    PRIMARY KEY (id)
);


--
-- Name: posts; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE posts (
    id SERIAL NOT NULL,
    title character varying(200) NOT NULL,
    content text,
    author_id integer,
    status public.status DEFAULT 'active'::status,
    created_at timestamp DEFAULT now(),
    PRIMARY KEY (id),
    FOREIGN KEY (author_id) REFERENCES users (id)
);


--
-- Name: idx_users_email; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_users_email ON users (email);
