ALTER TABLE users ADD COLUMN email email_address NOT NULL;

ALTER TABLE users ADD COLUMN status user_status DEFAULT 'active' NOT NULL;
