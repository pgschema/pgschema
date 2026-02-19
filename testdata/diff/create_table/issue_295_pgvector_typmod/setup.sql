-- Setup: Requires pgvector extension
-- This test is skipped in CI (embedded-postgres doesn't include pgvector).
-- To run manually, install pgvector and remove from skipListRequiresExtension.
CREATE EXTENSION IF NOT EXISTS vector;
