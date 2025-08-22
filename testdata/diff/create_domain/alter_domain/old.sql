CREATE DOMAIN user_rating AS integer
  CHECK (VALUE >= 1 AND VALUE <= 5);