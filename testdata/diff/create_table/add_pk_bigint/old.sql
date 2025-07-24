CREATE TABLE public.transactions (
    id bigint NOT NULL,
    amount numeric(15,2) NOT NULL,
    created_at timestamp with time zone
);