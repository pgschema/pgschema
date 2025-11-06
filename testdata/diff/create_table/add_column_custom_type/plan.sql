ALTER TABLE users ADD COLUMN email citext NOT NULL;

ALTER TABLE users ADD COLUMN description public.custom_text;

ALTER TABLE users ADD COLUMN status public.status_enum DEFAULT 'active';
