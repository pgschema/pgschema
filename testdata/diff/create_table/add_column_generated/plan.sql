ALTER TABLE merge_request
ADD COLUMN iid integer GENERATED ALWAYS AS (CAST(data ->> 'iid' AS int)) STORED CONSTRAINT pk_merge_request_iid PRIMARY KEY;

ALTER TABLE merge_request ADD COLUMN title text GENERATED ALWAYS AS (data ->> 'title') STORED;
