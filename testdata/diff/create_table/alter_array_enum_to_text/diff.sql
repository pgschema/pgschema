ALTER TABLE products ALTER COLUMN tags TYPE text[] USING tags::text[];
