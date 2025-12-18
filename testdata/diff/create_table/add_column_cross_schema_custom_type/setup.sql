-- Setup: Test type qualification for types in different schemas
-- This reproduces GitHub #197 and validates fixes for:
--
-- PUBLIC schema types (citext):
--   Users naturally write unqualified types when both table and type are in public.
--   pgschema must include public in search_path when applying to temp schema,
--   otherwise type resolution fails with "type citext does not exist".
--
-- CUSTOM schema types (hstore, custom_text, custom_enum):
--   Types in non-public schemas must always be schema-qualified in the output.

-- Drop and recreate utils schema for idempotency (setup runs for both old.sql and new.sql)
DROP SCHEMA IF EXISTS utils CASCADE;
CREATE SCHEMA utils;

-- Domain and enum in utils schema - must be schema-qualified
CREATE DOMAIN utils.custom_text AS text;
CREATE TYPE utils.custom_enum AS ENUM ('active', 'inactive', 'pending');

-- hstore in utils schema - tests cross-schema type qualification
CREATE EXTENSION IF NOT EXISTS hstore SCHEMA utils;

-- citext in public schema - reproduces #197
-- Users write unqualified "citext" when in public schema (natural usage)
CREATE EXTENSION IF NOT EXISTS citext SCHEMA public;
