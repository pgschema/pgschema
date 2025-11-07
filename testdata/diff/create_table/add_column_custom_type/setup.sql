-- Setup: Create types in public schema
-- This simulates extension types like citext installed in public schema

CREATE TYPE public.email_address AS (
    local_part text,
    domain text
);

CREATE TYPE public.user_status AS ENUM (
    'active',
    'inactive',
    'suspended',
    'pending'
);
