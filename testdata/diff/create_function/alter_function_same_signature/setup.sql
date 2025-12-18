-- Create utils schema with a custom type for cross-schema testing
-- Drop and recreate for idempotency (setup runs for both old.sql and new.sql)
DROP SCHEMA IF EXISTS utils CASCADE;
CREATE SCHEMA utils;

CREATE TYPE utils.priority_level AS ENUM ('low', 'medium', 'high', 'critical');
