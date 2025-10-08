CREATE TABLE public.users (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    email VARCHAR(100),
    bio TEXT,
    status VARCHAR(20) DEFAULT 'active'
);

CREATE VIEW public.user_search_view AS
SELECT
    id,
    COALESCE(first_name || ' ' || last_name, 'Anonymous') as display_name,
    COALESCE(email, '') as email,
    COALESCE(bio, 'No description available') as description,
    to_tsvector('english', COALESCE(first_name, '') || ' ' || COALESCE(last_name, '') || ' ' || COALESCE(bio, '')) as search_vector
FROM users
WHERE status = 'active';