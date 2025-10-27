CREATE OR REPLACE VIEW active_employees AS
SELECT 
    status,
    COUNT(*) as employee_count,
    AVG(salary) as avg_salary
FROM employees
WHERE status = 'active'
GROUP BY status
HAVING AVG(salary) > 50000
ORDER BY employee_count ASC, avg_salary DESC;
