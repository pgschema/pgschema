CREATE TABLE public.transactions (
    id bigint PRIMARY KEY,
    amount numeric(15,2) NOT NULL,
    created_at timestamp with time zone
);