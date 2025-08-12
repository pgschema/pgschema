--
-- Name: positive_integer; Type: DOMAIN; Schema: -; Owner: -
--

CREATE DOMAIN positive_integer AS integer
  CONSTRAINT positive_integer_check CHECK (VALUE > 0);
