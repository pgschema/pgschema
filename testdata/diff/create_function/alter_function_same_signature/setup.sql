-- Create utils schema with a custom type for cross-schema testing
CREATE SCHEMA IF NOT EXISTS utils;

CREATE TYPE utils.priority_level AS ENUM ('low', 'medium', 'high', 'critical');
