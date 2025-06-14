--
-- PostgreSQL database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 0.0.1


--
-- Name: orders; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.orders (
    id integer NOT NULL,
    order_date date DEFAULT CURRENT_DATE,
    total_amount numeric(12,2),
    status text DEFAULT 'pending'::text,
    CONSTRAINT order_date_valid CHECK ((order_date >= '2020-01-01'::date)),
    CONSTRAINT total_positive CHECK ((total_amount > (0)::numeric)),
    CONSTRAINT valid_status CHECK ((status = ANY (ARRAY['pending'::text, 'processing'::text, 'shipped'::text, 'delivered'::text, 'cancelled'::text])))

);

--
-- Name: orders_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.orders_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: orders_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.orders_id_seq OWNED BY public.orders.id;

--
-- Name: products; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.products (
    id integer NOT NULL,
    name text NOT NULL,
    price numeric(10,2),
    category text,
    stock integer,
    CONSTRAINT name_not_empty CHECK ((length(TRIM(BOTH FROM name)) > 0)),
    CONSTRAINT price_positive CHECK ((price > (0)::numeric)),
    CONSTRAINT stock_non_negative CHECK ((stock >= 0)),
    CONSTRAINT valid_category CHECK ((category = ANY (ARRAY['electronics'::text, 'books'::text, 'clothing'::text])))

);

--
-- Name: products_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.products_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--
-- Name: products_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.products_id_seq OWNED BY public.products.id;

--
-- Name: orders id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.orders ALTER COLUMN id SET DEFAULT nextval('public.orders_id_seq'::regclass);

--
-- Name: products id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.products ALTER COLUMN id SET DEFAULT nextval('public.products_id_seq'::regclass);

--
-- Name: orders orders_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.orders
    ADD CONSTRAINT orders_pkey PRIMARY KEY (id);

--
-- Name: products products_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.products
    ADD CONSTRAINT products_pkey PRIMARY KEY (id);

--
-- PostgreSQL database dump complete
--

