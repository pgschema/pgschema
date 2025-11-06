-- Setup: Create extension type, custom domain, and enum to test type qualification
-- This reproduces GitHub #144 and validates PR #145 fixes
-- Extension types (citext) should be unqualified (search_path resolution)
-- Custom domains and enums should be schema-qualified (public.*)

CREATE EXTENSION IF NOT EXISTS citext;

CREATE DOMAIN public.custom_text AS text;

CREATE TYPE public.status_enum AS ENUM ('active', 'inactive', 'pending');
