CREATE TABLE public.transactions (
    id bigint,
    amount numeric(15,2) NOT NULL,
    created_at timestamp with time zone,
    CONSTRAINT transactions_pkey PRIMARY KEY (id)
);