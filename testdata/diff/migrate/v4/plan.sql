CREATE OR REPLACE PROCEDURE simple_salary_update(
    p_emp_no integer,
    p_amount integer
)
LANGUAGE plpgsql
AS $$
BEGIN
    -- Simple update of salary amount
    UPDATE salary 
    SET amount = p_amount 
    WHERE emp_no = p_emp_no 
    AND to_date = '9999-01-01';
    
    RAISE NOTICE 'Updated salary for employee % to $%', p_emp_no, p_amount;
END;
$$;

CREATE OR REPLACE VIEW dept_emp_latest_date AS
 SELECT
    emp_no,
    max(from_date) AS from_date,
    max(to_date) AS to_date
   FROM dept_emp
  GROUP BY emp_no;

CREATE OR REPLACE VIEW current_dept_emp AS
 SELECT
    l.emp_no,
    d.dept_no,
    l.from_date,
    l.to_date
   FROM dept_emp d
     JOIN dept_emp_latest_date l ON d.emp_no = l.emp_no AND d.from_date = l.from_date AND l.to_date = d.to_date;

CREATE OR REPLACE TRIGGER salary_log_trigger
    AFTER UPDATE OR DELETE ON salary
    FOR EACH ROW
    EXECUTE FUNCTION log_dml_operations('payroll', 'high');

CREATE OR REPLACE FUNCTION log_dml_operations()
RETURNS trigger
LANGUAGE plpgsql
SECURITY INVOKER
VOLATILE
AS $$
DECLARE
    table_category TEXT;
    log_level TEXT;
BEGIN
    -- Get arguments passed from trigger (if any)
    -- TG_ARGV[0] is the first argument, TG_ARGV[1] is the second
    table_category := COALESCE(TG_ARGV[0], 'default');
    log_level := COALESCE(TG_ARGV[1], 'standard');
    
    IF (TG_OP = 'INSERT') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (
            'INSERT [' || table_category || ':' || log_level || ']', 
            current_query(), 
            current_user
        );
        RETURN NEW;
    ELSIF (TG_OP = 'UPDATE') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (
            'UPDATE [' || table_category || ':' || log_level || ']', 
            current_query(), 
            current_user
        );
        RETURN NEW;
    ELSIF (TG_OP = 'DELETE') THEN
        INSERT INTO audit (operation, query, user_name)
        VALUES (
            'DELETE [' || table_category || ':' || log_level || ']', 
            current_query(), 
            current_user
        );
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_operation ON audit (operation);

-- pgschema:wait
SELECT 
    COALESCE(i.indisvalid, false) as done,
    CASE 
        WHEN p.blocks_total > 0 THEN p.blocks_done * 100 / p.blocks_total
        ELSE 0
    END as progress
FROM pg_class c
LEFT JOIN pg_index i ON c.oid = i.indexrelid
LEFT JOIN pg_stat_progress_create_index p ON c.oid = p.index_relid
WHERE c.relname = 'idx_audit_operation';

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_username ON audit (user_name);

-- pgschema:wait
SELECT 
    COALESCE(i.indisvalid, false) as done,
    CASE 
        WHEN p.blocks_total > 0 THEN p.blocks_done * 100 / p.blocks_total
        ELSE 0
    END as progress
FROM pg_class c
LEFT JOIN pg_index i ON c.oid = i.indexrelid
LEFT JOIN pg_stat_progress_create_index p ON c.oid = p.index_relid
WHERE c.relname = 'idx_audit_username';
