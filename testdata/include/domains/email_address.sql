--
-- Name: email_address; Type: DOMAIN; Schema: -; Owner: -
--

CREATE DOMAIN email_address AS text
  CONSTRAINT email_address_check CHECK (VALUE ~~ '%@%');
