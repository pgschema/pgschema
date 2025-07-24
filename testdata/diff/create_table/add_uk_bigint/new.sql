CREATE TABLE public.transactions (
    id bigint UNIQUE,
    amount numeric(15,2) NOT NULL,
    created_at timestamp with time zone
);