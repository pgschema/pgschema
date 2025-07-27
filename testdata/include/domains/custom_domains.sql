CREATE DOMAIN email_address AS TEXT
    CHECK (VALUE LIKE '%@%');

CREATE DOMAIN positive_integer AS INTEGER
    CHECK (VALUE > 0);