ALTER TABLE merge_request
ADD COLUMN iid integer PRIMARY KEY GENERATED ALWAYS AS (CAST(data ->> 'iid' AS int)) STORED;

ALTER TABLE merge_request ADD COLUMN title text GENERATED ALWAYS AS (data ->> 'title') STORED;
