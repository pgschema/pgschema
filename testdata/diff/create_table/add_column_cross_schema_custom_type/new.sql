-- New state: Add columns using extension types, custom domain, and enum
-- Types are created via setup.sql
-- This tests GitHub #144 fix and PR #145 functionality:
--   - Extension types in external schemas (hstore) should be qualified
--   - Custom domains and enums in external schemas should be qualified

CREATE TABLE public.users (
    id bigint PRIMARY KEY,
    username text NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    -- Extension type from utils schema: must be qualified
    metadata utils.hstore,
    fqdn utils.citext NOT NULL,
    -- Custom domain from utils schema: must be qualified
    description utils.custom_text,
    -- Enum type from utils schema: must be qualified
    status utils.custom_enum DEFAULT 'active'
);