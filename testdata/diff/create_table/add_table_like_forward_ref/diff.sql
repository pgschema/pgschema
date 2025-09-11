CREATE TABLE IF NOT EXISTS customers (
    customer_id integer NOT NULL,
    name varchar(100) NOT NULL,
    email varchar(255) UNIQUE,
    created_at timestamp DEFAULT now(),
    updated_at timestamp DEFAULT now()
);

CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    order_date date NOT NULL,
    customer_id integer NOT NULL,
    name varchar(100) NOT NULL,
    email varchar(255),
    created_at timestamp DEFAULT now(),
    updated_at timestamp DEFAULT now()
);