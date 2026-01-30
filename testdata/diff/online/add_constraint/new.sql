CREATE TABLE public.orders (
    id integer NOT NULL,
    customer_id integer NOT NULL,
    amount numeric(10,2),
    status varchar(10) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT check_amount_positive CHECK (amount > 0),
    CONSTRAINT check_valid_status CHECK (status IN ('pending', 'shipped', 'delivered'))
);