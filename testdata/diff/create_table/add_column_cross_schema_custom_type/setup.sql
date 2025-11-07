-- Setup: Create extension type, custom domain, and enum to test type qualification
-- This reproduces GitHub #144 and validates PR #145 fixes
-- All types from external schemas (not the target schema) should be schema-qualified
-- This includes:
--   - Extension types (hstore)
--   - Custom domains and enums

CREATE SCHEMA IF NOT EXISTS utils;

CREATE DOMAIN utils.custom_text AS text;

CREATE TYPE utils.custom_enum AS ENUM ('active', 'inactive', 'pending');

-- hstore type stays in utils schema
CREATE EXTENSION IF NOT EXISTS hstore SCHEMA utils;
CREATE EXTENSION IF NOT EXISTS citext SCHEMA utils;