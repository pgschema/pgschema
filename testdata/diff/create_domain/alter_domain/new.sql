CREATE DOMAIN user_rating AS integer
  DEFAULT 3
  CHECK (VALUE >= 1 AND VALUE <= 10);