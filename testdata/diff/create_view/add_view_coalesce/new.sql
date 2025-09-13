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

CREATE VIEW public.user_search_view AS
SELECT 
    u.id,
    COALESCE(p.display_name, u.first_name || ' ' || u.last_name) as display_name,
    COALESCE(u.email, '') as email,
    COALESCE(p.description, u.bio, 'No description available') as description,
    to_tsvector('english', COALESCE(p.display_name, '') || ' ' || COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '') || ' ' || COALESCE(p.description, '')) as search_vector
FROM users u
LEFT JOIN profiles p ON u.id = p.user_id
WHERE u.status = 'active';