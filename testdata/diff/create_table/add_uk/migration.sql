ALTER TABLE users
ADD CONSTRAINT users_email_key UNIQUE (email);

ALTER TABLE users
ADD CONSTRAINT users_username_key UNIQUE (username);