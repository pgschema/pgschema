ALTER TABLE merge_request
ADD COLUMN iid integer GENERATED ALWAYS AS (((data ->> 'iid'::text))::integer) STORED CONSTRAINT pk_merge_request_iid PRIMARY KEY;

ALTER TABLE merge_request ADD COLUMN title text GENERATED ALWAYS AS ((data ->> 'title'::text)) STORED;

ALTER TABLE merge_request ADD COLUMN cleaned_title varchar(255) GENERATED ALWAYS AS (lower((data ->> 'title'::text))) STORED NOT NULL;