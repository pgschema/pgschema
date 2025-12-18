CREATE TYPE tag_type AS ENUM (
    'featured',
    'sale',
    'new',
    'popular'
);

ALTER TABLE products ALTER COLUMN tags TYPE tag_type[] USING tags::tag_type[];
