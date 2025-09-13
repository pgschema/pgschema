CREATE TABLE public.users (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    email VARCHAR(100),
    bio TEXT,
    status VARCHAR(20) DEFAULT 'active'
);

CREATE TABLE public.profiles (
    user_id INTEGER REFERENCES users(id),
    display_name VARCHAR(100),
    description TEXT,
    metadata JSONB
);