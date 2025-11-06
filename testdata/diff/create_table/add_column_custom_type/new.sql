-- New state: Add columns using extension type, custom domain, and enum
-- Types are created via setup.sql
-- This tests GitHub #144 fix and PR #145 functionality:
--   - Extension types (citext) should be unqualified
--   - Custom domains (custom_text) should be schema-qualified
--   - Enums (status_enum) should be schema-qualified

CREATE TABLE public.users (
    id bigint PRIMARY KEY,
    username text NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    -- Extension type: should be unqualified
    email citext NOT NULL,
    -- Custom domain: should be qualified as public.custom_text
    description custom_text,
    -- Enum type: should be qualified as public.status_enum
    status status_enum DEFAULT 'active'
);
