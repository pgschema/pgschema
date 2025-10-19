CREATE TYPE public.user_role AS ENUM ('admin', 'user');
CREATE TYPE public.status AS ENUM ('active', 'inactive');

--
-- Shared table in public schema that tenant schemas will reference
--
CREATE TABLE public.categories (
    id SERIAL PRIMARY KEY,
    name varchar(100) NOT NULL UNIQUE,
    description text
);
