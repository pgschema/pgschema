ALTER TABLE users ADD COLUMN email citext NOT NULL;

ALTER TABLE users ADD COLUMN description custom_text;

ALTER TABLE users ADD COLUMN status status_enum DEFAULT 'active';
