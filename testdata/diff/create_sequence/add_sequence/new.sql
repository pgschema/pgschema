CREATE SEQUENCE public.user_id_seq;

CREATE SEQUENCE public.order_seq INCREMENT BY 10 CYCLE;

CREATE SEQUENCE public.small_seq AS smallint CACHE 20;

CREATE SEQUENCE public.int_seq AS integer START WITH 100 CACHE 5;

CREATE SEQUENCE public.big_seq AS bigint MAXVALUE 1000000 CACHE 10;