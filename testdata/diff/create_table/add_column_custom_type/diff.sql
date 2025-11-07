ALTER TABLE users ADD COLUMN email utils.citext NOT NULL;
ALTER TABLE users ADD COLUMN description utils.custom_text;
ALTER TABLE users ADD COLUMN status utils.custom_enum DEFAULT 'active';