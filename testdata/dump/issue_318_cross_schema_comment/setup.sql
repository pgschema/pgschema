--
-- Setup: Two schemas with identically-named tables but different comments.
-- This reproduces GitHub issue #318 where the buggy pg_class join on relname
-- alone (without relnamespace) can cause wrong comment attribution.
--

CREATE SCHEMA alpha;
CREATE SCHEMA beta;

CREATE TABLE alpha.account (
    id serial PRIMARY KEY,
    name text NOT NULL
);

COMMENT ON TABLE alpha.account IS 'Alpha account table';
COMMENT ON COLUMN alpha.account.name IS 'Alpha account name';

CREATE TABLE beta.account (
    id serial PRIMARY KEY,
    name text NOT NULL
);

COMMENT ON TABLE beta.account IS 'Beta account table';
COMMENT ON COLUMN beta.account.name IS 'Beta account name';
