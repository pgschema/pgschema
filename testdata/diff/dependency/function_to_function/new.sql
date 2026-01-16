-- Base function that returns a simple type
CREATE OR REPLACE FUNCTION public.get_raw_result()
RETURNS integer
LANGUAGE SQL
RETURN 42;

-- Function with default value that references first function
-- PostgreSQL tracks this dependency via pg_depend
CREATE OR REPLACE FUNCTION public.process_result(val integer DEFAULT get_raw_result())
RETURNS text
LANGUAGE SQL
RETURN ('Processed: '::text || val::text);
