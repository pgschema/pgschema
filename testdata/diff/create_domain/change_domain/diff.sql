ALTER DOMAIN user_rating SET DEFAULT 3;

ALTER DOMAIN user_rating DROP CONSTRAINT user_rating_check;

ALTER DOMAIN user_rating ADD CONSTRAINT user_rating_check CHECK ((VALUE >= 1) AND (VALUE <= 10));
