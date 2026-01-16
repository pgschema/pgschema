CREATE OR REPLACE FUNCTION get_raw_result()
RETURNS integer
LANGUAGE sql
VOLATILE
RETURN 42;

CREATE OR REPLACE FUNCTION process_result(
    val integer DEFAULT get_raw_result()
)
RETURNS text
LANGUAGE sql
VOLATILE
RETURN ('Processed: '::text || (val)::text);
